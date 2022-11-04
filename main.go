package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/nelsonlai-go/args"
)

const OPENZEPPELIN_PACKAGE_NAME = "@openzeppelin"

var (
	ARGS                    *args.Args
	ACTION                  string
	FLAG_LANG               string
	FLAG_TARGET             string
	FLAG_PACKAGE_NAME       string
	FLAG_CONTRACT_FOLDER    string
	FLAG_NODE_MODULE_FOLDER string
	FLAG_OUTPUT_FOLDER      string
)

type Flags struct {
	Lang             string
	Target           string
	ContractFolder   string
	NodeModuleFolder string
	Output           string
	Package          string
}

func main() {
	ARGS = args.New()

	action := getAction()
	switch action {
	case "help":
		fmt.Println(`
=========
| Wrap3 |
=========

Generate wrapper class of solidity contract for Java & Golang.

Usage:

	wrap3 [OPTIONS] [COMMAND]

Commands:

	help	- get help

	compile - compile the solidity contract to wrapper class
	  options:
	    --lang (-l)      | required                            | java or go or abi, language to compile
	    --target (-t)    | required	                           | target contract file name (without extension)
	    --package (-p)   | required if lang is java or go      | the package name of the java class or golang file
	    --contracts (-c) | optional (default: ./contracts)     | the folder containing the .sol files
	    --node (-n)      | optional (default: ./node_modules)  | the node modules folder containing the @openzeppelin package
	    --output (-o)    | optional (default: ./wrap3)         | the output folder
		`)
	case "compile":
		parseFlags()
		switch FLAG_LANG {
		case "java":
			compileJava()
		case "go":
			compileGo()
		case "abi":
			compileABI()
		default:
			log.Fatalf("non-support lang: %s\n", FLAG_LANG)
		}
	default:
		log.Fatalf("unknown action: %s\n", action)
	}
}

func compileGo() {
	createTempFolder()
	defer removeTempFolder()

	copyContractFolder()
	copyOpenZeppelinPackage()

	processAllContractFiles()

	solcCompile()
	abigenCompile()
}

func compileJava() {
	createTempFolder()
	defer removeTempFolder()

	copyContractFolder()
	copyOpenZeppelinPackage()

	processAllContractFiles()

	solcCompile()
	web3jCompile()
}

func compileABI() {
	createTempFolder()
	defer removeTempFolder()

	copyContractFolder()
	copyOpenZeppelinPackage()

	processAllContractFiles()
	solcCompile()

	err := os.MkdirAll(FLAG_OUTPUT_FOLDER, os.ModePerm)
	if err != nil {
		panic(err)
	}

	cpAbiExec := exec.Command("cp", "./temp/artifacts/"+FLAG_TARGET+".abi", FLAG_OUTPUT_FOLDER+"/"+FLAG_TARGET+".abi")
	err = cpAbiExec.Run()
	if err != nil {
		log.Fatalln("Failed to copy .abi: ", cpAbiExec.String())
	}

	cpBinExec := exec.Command("cp", "./temp/artifacts/"+FLAG_TARGET+".bin", FLAG_OUTPUT_FOLDER+"/"+FLAG_TARGET+".bin")
	err = cpBinExec.Run()
	if err != nil {
		log.Fatalln("Failed to copy .abi: ", cpBinExec.String())
	}
}

func web3jCompile() {
	web3jExec := exec.Command("web3j", "generate", "solidity", "-b", "./temp/artifacts/"+FLAG_TARGET+".bin", "-a", "./temp/artifacts/"+FLAG_TARGET+".abi", "-o", FLAG_OUTPUT_FOLDER, "-p", FLAG_PACKAGE_NAME)
	err := web3jExec.Run()
	if err != nil {
		log.Fatalln("Failed to compile with web3j: ", web3jExec.String())
	}
}

func abigenCompile() {
	abigenExec := exec.Command("abigen", "--bin=./temp/artifacts/"+FLAG_TARGET+".bin", "--abi=./temp/artifacts/"+FLAG_TARGET+".abi", "--out="+FLAG_OUTPUT_FOLDER+"/"+FLAG_TARGET+".go", "--pkg="+FLAG_PACKAGE_NAME)
	err := abigenExec.Run()
	if err != nil {
		log.Fatalln("Failed to compile with abigen(go): ", abigenExec.String())
	}
}

func solcCompile() {
	solcExec := exec.Command("solc", "./temp/contracts/"+FLAG_TARGET+".sol", "--bin", "--abi", "--overwrite", "-o", "./temp/artifacts")
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

func copyOpenZeppelinPackage() {
	from := FLAG_NODE_MODULE_FOLDER + "/" + OPENZEPPELIN_PACKAGE_NAME
	to := "./temp/contracts/" + OPENZEPPELIN_PACKAGE_NAME
	cmd := exec.Command("cp", "-r", from, to)
	err := cmd.Run()
	if err != nil {
		log.Fatalf("failed to copy folder - from: %s to: %s (error: %s)\n", from, to, err.Error())
	}
}

func copyContractFolder() {
	from := FLAG_CONTRACT_FOLDER
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

func parseFlags() {
	FLAG_LANG = ARGS.FlagString("lang", true, "", "l")
	FLAG_TARGET = ARGS.FlagString("target", true, "", "t")
	FLAG_PACKAGE_NAME = ARGS.FlagString("package", false, "", "p")
	FLAG_CONTRACT_FOLDER = ARGS.FlagString("contract", false, "./contracts", "c")
	FLAG_NODE_MODULE_FOLDER = ARGS.FlagString("node", false, "./node_modules", "n")
	FLAG_OUTPUT_FOLDER = ARGS.FlagString("output", false, "./wrap3", "o")
	if (FLAG_LANG == "java" || FLAG_LANG == "go") && FLAG_PACKAGE_NAME == "" {
		log.Fatalf("flags: [%s] is required when [lang] is java or go.\n", "package")
	}
}

func getAction() string {
	if len(ARGS.Args) == 0 {
		log.Fatalln("[command] is missing. usage: wrap3 [OPTIONS] [COMMAND]")
	}
	return ARGS.Args[0]
}
