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
			cmd := exec.Command(processor.Executable, args...)

			// Направляем вывод только в файл
			cmd.Stdout = resultFile
			cmd.Stderr = resultFile

			err = cmd.Run()
			if err != nil {
				errMsg := fmt.Sprintf("\nCommand execution failed with error: %v\n", err)
				resultFile.WriteString(errMsg)
			}

			// Убрали вывод о успешном завершении в консоль
		}
	}

	return nil
}

/// END MODULE
