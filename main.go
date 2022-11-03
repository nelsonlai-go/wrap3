package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

/*
	wrap3 help

	wrap3
		-lang[-l] <java|go>
		-target[-t] <contracts/Contract.sol>
		-contract-folder[-cf] <contracts> (optional)
		-node-module-folder[-nf] <node-modules> (optional)
		-output[-o] <./output> (optional)
		-package[-p] <package>
		compile
*/

const OPENZEPPELIN_PACKAGE_NAME = "@openzeppelin"

type Flags struct {
	Lang             *string
	Target           *string
	ContractFolder   *string
	NodeModuleFolder *string
	Output           *string
	Package          *string
}

func main() {
	action := getAction()
	switch action {
	case "help":
		fmt.Println(`
wrap3 help

wrap3
 -l <java | go | abi> [language of wrapper class to compile]
 -t <Contract> [target of contract to compile, file extension is no needed]
 -cf <contracts> (optional, default: ./contracts) [contract folder of .sol files] 
 -nf <node-modules> (optional, default: ./node-modules) [app will find @openzeppelin package in this folder]
 -o <./output> (optional, default: ./wrap3) [output folder]
 -p <package> [package name of the java class or the go file]
 compile`)
	case "compile":
		fs := parseFlags()
		switch *fs.Lang {
		case "java":
			compileJava(fs)
		case "go":
			compileGo(fs)
		case "abi":
			compileABI(fs)
		default:
			log.Fatalf("non-support lang: %s\n", *fs.Lang)
		}
	default:
		log.Fatalf("unknown action: %s\n", action)
	}
}

func compileGo(fs *Flags) {
	createTempFolder()
	defer removeTempFolder()

	copyContractFolder(fs)
	copyOpenZeppelinPackage(fs)

	processAllContractFiles()

	solcCompile(fs)
	abigenCompile(fs)
}

func compileJava(fs *Flags) {
	createTempFolder()
	defer removeTempFolder()

	copyContractFolder(fs)
	copyOpenZeppelinPackage(fs)

	processAllContractFiles()

	solcCompile(fs)
	web3jCompile(fs)
}

func compileABI(fs *Flags) {
	createTempFolder()
	defer removeTempFolder()

	copyContractFolder(fs)
	copyOpenZeppelinPackage(fs)

	processAllContractFiles()
	solcCompile(fs)

	cpExec := exec.Command("cp", "./temp/artifacts/"+*fs.Target+".bin", *fs.Output+"/"+*fs.Target+".bin")
	err := os.MkdirAll(*fs.Output, os.ModePerm)
	if err != nil {
		panic(err)
	}
	err = cpExec.Run()
	if err != nil {
		log.Fatalln("Failed to copy .abi: ", cpExec.String())
	}
}

func web3jCompile(fs *Flags) {
	web3jExec := exec.Command("web3j", "generate", "solidity", "-b", "./temp/artifacts/"+*fs.Target+".bin", "-a", "./temp/artifacts/"+*fs.Target+".abi", "-o", *fs.Output, "-p", *fs.Package)
	err := web3jExec.Run()
	if err != nil {
		log.Fatalln("Failed to compile with web3j: ", web3jExec.String())
	}
}

func abigenCompile(fs *Flags) {
	abigenExec := exec.Command("abigen", "--bin=./temp/artifacts/"+*fs.Target+".bin", "--abi=./temp/artifacts/"+*fs.Target+".abi", "--out="+*fs.Output+"/"+*fs.Target+".go", "--pkg="+*fs.Package)
	err := abigenExec.Run()
	if err != nil {
		log.Fatalln("Failed to compile with abigen(go): ", abigenExec.String())
	}
}

func solcCompile(fs *Flags) {
	solcExec := exec.Command("solc", "./temp/contracts/"+*fs.Target+".sol", "--bin", "--abi", "--overwrite", "-o", "./temp/artifacts")
	err := solcExec.Run()
	if err != nil {
		log.Fatalln("Failed to compile with solc: ", solcExec.String())
	}
}

func processAllContractFiles() {
	contractFilePaths := getAllContractFilePaths("./temp/contracts")
	for _, path := range contractFilePaths {
		readContractFileAndReplaceImports(path)
	}
}

func readContractFileAndReplaceImports(path string) {
	content, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}

	txt := string(content)
	txt = strings.ReplaceAll(txt, `import "@openzeppelin`, `import "./@openzeppelin`)

	err = os.WriteFile(path, []byte(txt), os.ModePerm)
	if err != nil {
		panic(err)
	}
}

func getAllContractFilePaths(dir string) []string {
	paths := make([]string, 0)

	ds, err := os.ReadDir(dir)
	if err != nil {
		panic(err)
	}

	for _, d := range ds {
		if d.IsDir() {
			paths = append(paths, getAllContractFilePaths(dir+"/"+d.Name())...)
		} else if strings.Contains(d.Name(), ".sol") {
			paths = append(paths, dir+"/"+d.Name())
		}
	}

	return paths
}

func copyOpenZeppelinPackage(fs *Flags) {
	from := *fs.NodeModuleFolder + "/" + OPENZEPPELIN_PACKAGE_NAME
	to := "./temp/contracts/" + OPENZEPPELIN_PACKAGE_NAME
	cmd := exec.Command("cp", "-r", from, to)
	err := cmd.Run()
	if err != nil {
		log.Fatalf("failed to copy folder - from: %s to: %s (error: %s)\n", from, to, err.Error())
	}
}

func copyContractFolder(fs *Flags) {
	from := *fs.ContractFolder
	to := "./temp"
	err := os.MkdirAll(to, os.ModePerm)
	if err != nil {
		panic(err)
	}
	cmd := exec.Command("cp", "-r", from, to)
	err = cmd.Run()
	if err != nil {
		log.Fatalf("failed to copy folder - from: %s to: %s (error: %s)\n", from, to, err.Error())
	}
}

func createTempFolder() {
	removeTempFolder()
	err := os.MkdirAll("./temp", os.ModePerm)
	if err != nil {
		panic(err)
	}
}

func removeTempFolder() {
	err := os.RemoveAll("./temp")
	if err != nil {
		panic(err)
	}
}

func parseFlags() *Flags {
	f := &Flags{
		Lang:             flag.String("l", "", "language"),
		Target:           flag.String("t", "", "target"),
		ContractFolder:   flag.String("cf", "./contracts", "contract folder"),
		NodeModuleFolder: flag.String("nf", "./node_modules", "node module folder"),
		Output:           flag.String("o", "./wrap3", "output folder"),
		Package:          flag.String("p", "", "package name"),
	}
	flag.Parse()

	if *f.Lang == "" || *f.Target == "" {
		log.Fatalln("-l, -t are required")
	}

	if (*f.Lang == "java" || *f.Lang == "go") && *f.Package == "" {
		log.Fatalln("-p is required")
	}
	return f
}

func getAction() string {
	if len(os.Args) < 2 {
		log.Fatalln("action is missing. usage: wrap3 <action> <options>")
	}
	return os.Args[len(os.Args)-1]
}
