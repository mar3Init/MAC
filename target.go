package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v2"
)

// Обьединенные шаблоны, которые содержат в себе большое количество других маленьких шаблонов
type ComproudTarget struct {
	Description         string               `yaml:"Description"`
	Author              string               `yaml:"Author"`
	Version             string               `yaml:"Version"`
	Id                  string               `yaml:"Id"`
	RecreateDirectories bool                 `yaml:"RecreateDirectories"`
	Targets             []ComproudTargetInfo `yaml:"Targets"`
}

type ComproudTargetInfo struct {
	Name     string `yaml:"Name"`
	Category string `yaml:"Category"`
	Path     string `yaml:"Path"`
}

// Target представляет структуру шаблона, который содержит в себе команды для сбора
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

// Тип для кеширования результатов поиска
type searchCacheType struct {
	sync.RWMutex
	cache map[string][]string
}

var searchCacheInstance = &searchCacheType{
	cache: make(map[string][]string),
}

// loadTargets оптимизированная версия совместимая с Go 1.10.3
func (c *Collector) loadTargets() ([]Target, error) {
	var targets []Target
	targetNameMap := make(map[string]bool) // Используем map вместо массива для эффективной проверки

	// Шаг 1: Сканируем директорию на наличие Comproud-шаблонов
	var comproudPaths []string
	var regularPaths []string

	err := filepath.Walk(c.config.TargetsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Пропускаем ошибки доступа и продолжаем
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".tkape") {
			if strings.HasPrefix(info.Name(), "!") {
				if SearchUsingMap(c.config.needTargetName, strings.Replace(info.Name(), ".tkape", "", 1)) {
					comproudPaths = append(comproudPaths, path)
				}
			} else {
				regularPaths = append(regularPaths, path)
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Шаг 2: Обработка Comproud шаблонов и сбор целевых имен
	for _, path := range comproudPaths {
		data, err := ioutil.ReadFile(path)
		if err != nil {
			continue
		}

		var comproudTarget ComproudTarget
		if err := yaml.Unmarshal(data, &comproudTarget); err != nil {
			continue
		}

		for i := range comproudTarget.Targets {
			targetData := &comproudTarget.Targets[i]
			targetDataName := strings.Replace(targetData.Path, ".tkape", "", 1)
			targetNameMap[targetDataName] = true
			c.config.needTargetName = append(c.config.needTargetName, targetDataName)
		}
	}

	// Удаление дубликатов из c.config.needTargetName
	c.config.needTargetName = removeDuplicateStr(c.config.needTargetName)

	// Добавляем существующие имена в map
	for _, name := range c.config.needTargetName {
		targetNameMap[name] = true
	}

	// Шаг 3: Загрузка обычных шаблонов с использованием горутин и каналов
	var wg sync.WaitGroup
	resultChan := make(chan Target, len(regularPaths))
	semaphore := make(chan struct{}, 10) // Ограничиваем число одновременных горутин

	for _, path := range regularPaths {
		// Получаем имя без расширения для проверки
		filename := filepath.Base(path)
		targetName := strings.Replace(filename, ".tkape", "", 1)

		// Проверяем, нужен ли этот шаблон
		if !targetNameMap[targetName] {
			continue
		}

		wg.Add(1)
		semaphore <- struct{}{} // Блокируем, если достигнут предел

		go func(filePath string) {
			defer func() {
				<-semaphore // Освобождаем семафор
				wg.Done()
			}()

			data, err := ioutil.ReadFile(filePath)
			if err != nil {
				return
			}

			var target Target
			if err := yaml.Unmarshal(data, &target); err != nil {
				return
			}

			// Нормализация масок файлов
			for i := range target.Targets {
				targetData := &target.Targets[i]

				//if strings.Contains(targetData.Path, ".tkape") {
				//	data, err := ioutil.ReadFile(filePath)
				//	if err != nil {
				//		return
				//	}
				//
				//	var target Target
				//	if err := yaml.Unmarshal(data, &target); err != nil {
				//		return
				//	}
				//
				//} else {
				if strings.HasSuffix(targetData.FileMask, "X") {
					targetData.FileMask = strings.Replace(targetData.FileMask, "X", "*", 1)
				}
				if targetData.FileMask == "" && targetData.Recursive {
					targetData.FileMask = "*"
				}
			}

			//}

			//добавляем разбор Compound

			resultChan <- target
		}(path)
	}

	// Запускаем горутину для закрытия канала после завершения всех горутин
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Собираем результаты из канала
	for target := range resultChan {
		targets = append(targets, target)
	}

	return targets, nil
}

// CollectTargets оптимизированная версия для Go 1.10.3
func (c *Collector) CollectTargets() error {
	// Используем семафор для ограничения параллельной обработки таргетов
	targetSemaphore := make(chan struct{}, 4) // Ограничиваем максимум 4 таргета одновременно
	var targetWg sync.WaitGroup

	for _, target := range c.targets {
		targetWg.Add(1)
		targetSemaphore <- struct{}{}

		go func(t Target) {
			defer func() {
				<-targetSemaphore
				targetWg.Done()
			}()

			fmt.Printf("Start target %s\n", t.Description)
			c.processTarget(t)
			fmt.Printf("End target %s\n", t.Description)
		}(target)
	}

	targetWg.Wait()
	return nil
}

// processTarget заменяет copyWorker и оптимизирована для Go 1.10.3
func (c *Collector) processTarget(target Target) {
	// Собираем все пути к файлам из таргета без дубликатов
	fileMap := make(map[string]struct{})

	// Семафор для ограничения количества горутин поиска
	searchSemaphore := make(chan struct{}, 8)
	var searchWg sync.WaitGroup

	// Мьютекс для защиты доступа к fileMap
	var fileMapMutex sync.Mutex

	// Для каждого правила в таргете запускаем поиск
	for i := range target.Targets {
		searchWg.Add(1)
		searchSemaphore <- struct{}{}

		go func(idx int) {
			defer func() {
				<-searchSemaphore
				searchWg.Done()
			}()

			targetData := &target.Targets[idx]
			path := targetData.Path

			// Обработка специальных путей
			if strings.Contains(path, "%user%") {
				path = strings.Replace(path, "%user%", "*", 1)
			}

			// Если файл всегда нужно добавлять в очередь
			if targetData.AlwaysAddToQueue {
				fullPath := path + targetData.FileMask
				fileMapMutex.Lock()
				fileMap[fullPath] = struct{}{}
				fileMapMutex.Unlock()
			}

			// Кешированный поиск файлов
			files, _ := cachedSearchFiles(SearchOptions{
				Pattern:   targetData.FileMask,
				Path:      path,
				Recursive: targetData.Recursive,
			})

			// Добавляем найденные файлы в map
			if len(files) > 0 {
				fileMapMutex.Lock()
				for _, file := range files {
					fileMap[file] = struct{}{}
				}
				fileMapMutex.Unlock()
			}
		}(i)
	}

	// Ждем завершения всех поисковых операций
	searchWg.Wait()

	// Преобразуем map в slice
	var fileList []string
	for file := range fileMap {
		fileList = append(fileList, file)
	}

	// Копируем файлы параллельно с ограничением
	ch := make(chan int, c.config.MaxParallel)
	var copyWg sync.WaitGroup

	for _, pathCopy := range fileList {
		copyWg.Add(1)
		ch <- 1

		go func(path string) {
			defer func() {
				copyWg.Done()
				<-ch
			}()

			// Определяем путь для сохранения
			lastSlashIndex := strings.LastIndex(path, "\\")
			pathToSave := path
			if lastSlashIndex != -1 {
				pathToSave = path[:lastSlashIndex] // Оставляем строку до последнего '\'
			}

			// Копируем файл
			err := c.rawCopy.fullCopy(path, c.config.OutputPath+"\\"+strings.Replace(pathToSave, "C:", "C", 1))
			if err != nil {
				fmt.Println(err)
			}
		}(pathCopy)
	}

	copyWg.Wait()
}

// Кешированный поиск файлов
func cachedSearchFiles(opts SearchOptions) ([]string, error) {
	// Создаем ключ кеша
	cacheKey := fmt.Sprintf("%s|%s|%v", opts.Path, opts.Pattern, opts.Recursive)

	// Проверяем кеш
	searchCacheInstance.RLock()
	cached, found := searchCacheInstance.cache[cacheKey]
	searchCacheInstance.RUnlock()

	if found {
		return cached, nil
	}

	// Если нет в кеше, выполняем поиск
	results, err := searchFiles(opts)
	if err != nil {
		return results, err
	}

	// Кешируем результаты
	searchCacheInstance.Lock()
	searchCacheInstance.cache[cacheKey] = results
	searchCacheInstance.Unlock()

	return results, nil
}

// clearSearchCache очищает кеш поиска
func clearSearchCache() {
	searchCacheInstance.Lock()
	searchCacheInstance.cache = make(map[string][]string)
	searchCacheInstance.Unlock()
}

// removeDuplicateStr удаляет дубликаты из среза строк
func removeDuplicateStr(strSlice []string) []string {
	allKeys := make(map[string]bool)
	list := []string{}
	for _, item := range strSlice {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			list = append(list, item)
		}
	}
	return list
}
