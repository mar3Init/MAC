package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

// Module представляет структуру для выполнения команд
type Module struct {
	Description  string      `yaml:"Description"`
	Category     string      `yaml:"Category"`
	Author       string      `yaml:"Author"`
	Version      string      `yaml:"Version"`
	Id           string      `yaml:"Id"`
	BinaryUrl    string      `yaml:"BinaryUrl"`
	ExportFormat string      `yaml:"ExportFormat"`
	FileMask     string      `yaml:"FileMask"`
	Processors   []Processor `yaml:"Processors"`
}

type Processor struct {
	Executable   string `yaml:"Executable"`
	CommandLine  string `yaml:"CommandLine"`
	ExportFormat string `yaml:"ExportFormat"`
	ExportFile   string `yaml:"ExportFile"`
}

///MODULE

func (c *Collector) loadModules() ([]Module, error) {
	var modules []Module
	err := filepath.Walk(c.config.ModulesPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".mkape") {
			data, err := ioutil.ReadFile(path)
			if err != nil {
				return fmt.Errorf("error reading module file %s: %v", info.Name(), err)
			}

			var module Module
			if err := yaml.Unmarshal(data, &module); err != nil {
				return fmt.Errorf("error parsing target file %s: %v", info.Name(), err)
			}

			if SearchUsingMap(c.config.needModulesName, strings.Replace(info.Name(), ".mkape", "", 1)) {
				modules = append(modules, module)
			}

		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return modules, nil
}

func (c *Collector) ExecuteModules() error {
	for i := range c.modules {
		module := &c.modules[i]
		for j := range module.Processors {
			processor := &module.Processors[j]
			cmd := exec.Command(processor.Executable)
			err := cmd.Run()
			errMsg := fmt.Sprintf("%v", err)
			if strings.Contains(errMsg, "PATH") {
				c.ExecuteLocalModules(module, processor)
			} else {
				c.ExecuteGlobalModules(module, processor)
			}
		}
	}

	return nil
}
func (c *Collector) ExecuteGlobalModules(module *Module, processor *Processor) error {
	// Создаем выходную директорию для модуля
	outputPath := filepath.Join(c.config.OutputPath, "Module", module.Category)

	processor.CommandLine = strings.Replace(processor.CommandLine, "%kapedirectory%", ".", 1)
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return fmt.Errorf("error creating module output directory: %v", err)
	}

	// Подготавливаем команду
	cmdLine := strings.Replace(processor.CommandLine, "%sourceDirectory%", c.config.SourcePath, -1)
	cmdLine = strings.Replace(cmdLine, "%destinationDirectory%", outputPath, -1)

	// Правильная обработка командной строки для PowerShell
	var cmd *exec.Cmd
	if strings.Contains(processor.Executable, "powershell.exe") {
		// Для PowerShell передаем -Command как отдельный аргумент, а затем всю команду целиком
		cmd = exec.Command(processor.Executable, "-Command", cmdLine[10:len(cmdLine)-1]) // Удаляем внешние кавычки и префикс -Command
	} else {
		// Для других исполняемых файлов используем обычное разделение
		args := strings.Fields(cmdLine)
		cmd = exec.Command(processor.Executable, args...)
	}

	if processor.ExportFile != "" {
		resultFileName := fmt.Sprintf("%s", processor.ExportFile)
		resultFilePath := filepath.Join(outputPath, resultFileName)
		resultFile, err := os.OpenFile(resultFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("error creating result file: %v", err)
		}
		defer resultFile.Close()

		cmd.Stdout = resultFile
		cmd.Stderr = resultFile
	}

	err := cmd.Run()
	if err != nil {
		errMsg := fmt.Sprintf("\nCommand execution failed with error: %v\n", err)
		fmt.Printf(errMsg)
	}

	// Убрали вывод о успешном завершении в консоль
	return nil
}

func (c *Collector) ExecuteLocalModules(module *Module, processor *Processor) error {
	// Создаем выходную директорию для модуля
	outputPath := filepath.Join(c.config.OutputPath, "Module", module.Category)

	processor.CommandLine = strings.Replace(processor.CommandLine, "%kapedirectory%", ".", 1)
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return fmt.Errorf("error creating module output directory: %v", err)
	}

	// Создаем файл для записи результата
	resultFileName := fmt.Sprintf("%s", processor.ExportFile)
	resultFilePath := resultFileName
	if resultFileName == "" {
		resultFilePath = filepath.Join(outputPath, module.Description[:30])
	} else {
		resultFilePath = filepath.Join(outputPath, resultFileName)
	}

	resultFile, err := os.Create(resultFilePath)
	if err != nil {
		return fmt.Errorf("error creating result file: %v", err)
	}
	defer resultFile.Close()

	// Подготавливаем команду
	cmdLine := strings.Replace(processor.CommandLine, "%sourceDirectory%", c.config.SourcePath, -1)
	cmdLine = strings.Replace(cmdLine, "%destinationDirectory%", outputPath, -1)

	args := strings.Fields(cmdLine)
	cmd := exec.Command(filepath.Join(".\\", c.config.PathToAllTools, processor.Executable), args...)

	// Направляем вывод только в файл
	cmd.Stdout = resultFile
	cmd.Stderr = resultFile

	err = cmd.Run()
	if err != nil {
		errMsg := fmt.Sprintf("\nCommand execution failed with error: %v\n", err)
		resultFile.WriteString(errMsg)
	}

	// Убрали вывод о успешном завершении в консоль
	return nil
}

/// END MODULE
//
//
//
//func (c *Collector) ExecuteGlobalModules(module *Module, processor *Processor) error {
//	// Создаем выходную директорию для модуля
//	outputPath := filepath.Join(c.config.OutputPath, "Module", module.Category)
//
//	processor.CommandLine = strings.Replace(processor.CommandLine, "%kapedirectory%", ".", 1)
//	if err := os.MkdirAll(outputPath, 0755); err != nil {
//		return fmt.Errorf("error creating module output directory: %v", err)
//	}
//
//	// Создаем файл для записи результата
//	resultFileName := fmt.Sprintf("%s", processor.ExportFile)
//	resultFilePath := resultFileName
//	if resultFileName == "" {
//		if len(module.Description) > 30 {
//			resultFilePath = filepath.Join(outputPath, module.Description[:30])
//		} else {
//			resultFilePath = filepath.Join(outputPath, module.Description)
//		}
//	} else {
//		resultFilePath = filepath.Join(outputPath, resultFileName)
//	}
//
//	resultFile, err := os.Create(resultFilePath + ".txt")
//	if err != nil {
//		return fmt.Errorf("error creating result file: %v", err)
//	}
//	defer resultFile.Close()
//
//	// Подготавливаем команду
//	cmdLine := strings.Replace(processor.CommandLine, "%sourceDirectory%", c.config.SourcePath, -1)
//	cmdLine = strings.Replace(cmdLine, "%destinationDirectory%", outputPath, -1)
//
//	args := strings.Fields(cmdLine)
//	cmd := exec.Command(processor.Executable, args...)
//
//	// Направляем вывод только в файл
//	cmd.Stdout = resultFile
//	cmd.Stderr = resultFile
//
//	err = cmd.Run()
//	if err != nil {
//		errMsg := fmt.Sprintf("\nCommand execution failed with error: %v\n", err)
//		resultFile.WriteString(errMsg)
//	}
//
//	// Убрали вывод о успешном завершении в консоль
//	return nil
//}

//Wok, но не работает со сложными командами
