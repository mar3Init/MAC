package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

// Target представляет структуру для сбора артефактов
type Target struct {
	Description         string       `yaml:"Description"`
	Author              string       `yaml:"Author"`
	Version             string       `yaml:"Version"`
	Id                  string       `yaml:"Id"`
	RecreateDirectories bool         `yaml:"RecreateDirectories"`
	Targets             []TargetInfo `yaml:"Targets"`
}

type TargetInfo struct {
	Name             string `yaml:"Name"`
	Category         string `yaml:"Category"`
	Path             string `yaml:"Path"`
	Recursive        bool   `yaml:"Recursive"`
	FileMask         string `yaml:"FileMask"`
	AlwaysAddToQueue bool   `yaml:"AlwaysAddToQueue"`
	SaveAsFileName   string `yaml:"SaveAsFileName"`
	MinSize          int64  `yaml:"MinSize"`
	MaxSize          int64  `yaml:"MaxSize"`
	Comment          string `yaml:"Comment"`
	RawCopy          string
}

//TARGET

func (c *Collector) loadTargets() ([]Target, error) {
	var targets []Target

	err := filepath.Walk(c.config.TargetsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".tkape") {
			data, err := ioutil.ReadFile(path)
			if err != nil {
				return fmt.Errorf("error reading target file %s: %v", info.Name(), err)
			}

			var target Target
			if err := yaml.Unmarshal(data, &target); err != nil {
				return fmt.Errorf("error parsing target file %s: %v", info.Name(), err)
			}

			for i := range target.Targets {
				targetData := &target.Targets[i]
				if strings.HasSuffix(targetData.FileMask, "X") {
					targetData.FileMask = strings.Replace(targetData.FileMask, "X", "*", 1)
				}
				if targetData.FileMask == "" && targetData.Recursive {
					targetData.FileMask = "*"
				}
			}

			if SearchUsingMap(c.config.needTargetName, strings.Replace(info.Name(), ".tkape", "", 1)) {
				targets = append(targets, target)
			}

		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return targets, nil
}

func (c *Collector) CollectTargets() error {
	for _, target := range c.targets {
		fmt.Printf("Start target %s\n", target.Description)
		c.copyWorker(target)
		fmt.Printf("End target %s\n", target.Description)
	}
	c.wg.Wait()
	return nil
}

func (c *Collector) copyWorker(target Target) {

	FileList := []string{}
	data := []string{}

	for i := range target.Targets {
		targetData := &target.Targets[i]
		if strings.Contains(targetData.Path, "%user%") {
			targetData.Path = strings.Replace(targetData.Path, "%user%", "*", 1)
		}
		if targetData.AlwaysAddToQueue {
			FileList = append(FileList, targetData.Path+targetData.FileMask)
		}

		data, _ = searchFiles(SearchOptions{
			Pattern:   targetData.FileMask,
			Path:      targetData.Path,
			Recursive: targetData.Recursive,
		})
		for _, dataOne := range data {
			FileList = append(FileList, dataOne)
		}
	}

	ch := make(chan int, c.config.MaxParallel)
	for _, pathCopy := range FileList {
		c.wg.Add(1)
		ch <- 1
		go func(pathCopy string) {
			defer func() { c.wg.Done(); <-ch }()

			lastSlashIndex := strings.LastIndex(pathCopy, "\\")
			pathToSave := pathCopy
			if lastSlashIndex != -1 {
				pathToSave = pathCopy[:lastSlashIndex] // Оставляем строку до последнего '\'
			} else {
				pathToSave = pathCopy
			}
			err := c.rawCopy.fullCopy(pathCopy, c.config.OutputPath+"\\"+strings.Replace(pathToSave, "C:", "C", 1))
			if err != nil {
				fmt.Println(err)
			}
		}(pathCopy)
	}
	c.wg.Wait()
}

//END TARGET
