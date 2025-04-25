package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

//Поиск файлов

// SearchOptions содержит параметры поиска
type SearchOptions struct {
	Pattern   string
	Path      string
	Recursive bool
}

// Оптимизированная функция isMatch
func isMatch(pattern, name string) bool {
	// Оптимизация для случаев без wildcard
	if !strings.Contains(pattern, "*") {
		return pattern == name
	}

	// Для простых случаев с одной звездочкой в начале или конце
	if pattern == "*" {
		return true
	}

	if strings.HasPrefix(pattern, "*") && !strings.Contains(pattern[1:], "*") {
		return strings.HasSuffix(name, pattern[1:])
	}

	if strings.HasSuffix(pattern, "*") && !strings.Contains(pattern[:len(pattern)-1], "*") {
		return strings.HasPrefix(name, pattern[:len(pattern)-1])
	}

	// Разбиваем шаблон на части по *
	parts := strings.Split(pattern, "*")

	// Проверяем префикс
	if parts[0] != "" && !strings.HasPrefix(name, parts[0]) {
		return false
	}

	// Проверяем суффикс
	if parts[len(parts)-1] != "" && !strings.HasSuffix(name, parts[len(parts)-1]) {
		return false
	}

	// Для случаев с несколькими звездочками
	current := name
	for _, part := range parts {
		if part == "" {
			continue
		}

		index := strings.Index(current, part)
		if index == -1 {
			return false
		}

		current = current[index+len(part):]
	}

	return true
}

// searchFiles выполняет поиск файлов с поддержкой wildcard в путях
func searchFiles(opts SearchOptions) ([]string, error) {
	resultFiles := []string{}

	// Разбиваем путь на компоненты
	pathParts := strings.Split(filepath.Clean(opts.Path), string(os.PathSeparator))

	// Находим базовый путь (до первого wildcard)
	basePath := ""
	wildcardStartIndex := -1

	for i, part := range pathParts {
		if strings.Contains(part, "*") || strings.Contains(part, "?") {
			wildcardStartIndex = i
			break
		}
		if i > 0 {
			basePath = filepath.Join(basePath, "\\", part)
		} else {
			basePath = part
		}
	}

	// Если нет wildcards в пути, используем оригинальную логику
	if wildcardStartIndex == -1 {
		return searchFilesInDirectory(opts)
	}

	// Собираем оставшийся шаблон пути
	remainingPattern := strings.Join(pathParts[wildcardStartIndex:], string("\\"))

	// Функция для рекурсивного обхода директорий с wildcards
	var searchWithWildcards func(currentPath string, remainingParts []string) error
	searchWithWildcards = func(currentPath string, remainingParts []string) error {
		// Если больше нет частей пути для обработки, ищем файлы
		if len(remainingParts) == 0 {
			results, err := searchFilesInDirectory(SearchOptions{
				Path:      currentPath,
				Pattern:   opts.Pattern,
				Recursive: opts.Recursive,
			})
			if err == nil {
				resultFiles = append(resultFiles, results...)
			}
			return nil
		}

		// Получаем текущий шаблон
		currentPattern := remainingParts[0]

		// Читаем содержимое текущей директории
		entries, err := ioutil.ReadDir(currentPath)
		if err != nil {
			return nil
		}

		// Проверяем каждую запись на соответствие шаблону
		for _, entry := range entries {
			match, err := filepath.Match(currentPattern, entry.Name())
			if err != nil {
				continue
			}

			if match && entry.IsDir() {
				// Рекурсивно обрабатываем поддиректории
				nextPath := filepath.Join(currentPath, entry.Name())
				searchWithWildcards(nextPath, remainingParts[1:])
			}
		}
		return nil
	}

	// Начинаем поиск с базового пути
	err := searchWithWildcards(basePath, strings.Split(remainingPattern, string(os.PathSeparator)))
	if err != nil {
		return resultFiles, err
	}

	return resultFiles, nil
}

// searchFilesInDirectory выполняет поиск файлов в конкретной директории
func searchFilesInDirectory(opts SearchOptions) ([]string, error) {
	resultFiles := []string{}

	if !opts.Recursive {
		// Нерекурсивный поиск
		entries, err := ioutil.ReadDir(opts.Path)
		if err != nil {
			return resultFiles, nil
		}

		for _, entry := range entries {
			if isMatch(opts.Pattern, entry.Name()) && !entry.IsDir() {
				fullPath := filepath.Join(opts.Path, entry.Name())
				resultFiles = append(resultFiles, fullPath)
			}
		}
		return resultFiles, nil
	}

	// Рекурсивный поиск
	_ = filepath.Walk(opts.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if isMatch(opts.Pattern, info.Name()) && !info.IsDir() {
			resultFiles = append(resultFiles, path)
		}
		return nil
	})

	return resultFiles, nil
}

// SearchUsingMap - поиск с использованием map для большей производительности
func SearchUsingMap(arr []string, target string) bool {
	// Создаем map из массива для O(1) поиска
	lookup := make(map[string]struct{}, len(arr))
	for _, str := range arr {
		lookup[str] = struct{}{}
	}

	_, exists := lookup[target]
	return exists
}

//// searchFiles выполняет поиск файлов в соответствии с заданными параметрами
//func searchFiles(opts SearchOptions) ([]string, error) {
//	resultFiles := []string{}
//
//	if !opts.Recursive {
//		// Нерекурсивный поиск
//		entries, err := ioutil.ReadDir(opts.Path)
//		if err != nil {
//			return resultFiles, nil
//		}
//
//		for _, entry := range entries {
//			if isMatch(opts.Pattern, entry.Name()) && !entry.IsDir() {
//				fullPath := filepath.Join(opts.Path, entry.Name())
//				//fmt.Println(fullPath)
//				resultFiles = append(resultFiles, fullPath)
//			}
//		}
//		return resultFiles, nil
//	}
//
//	// Рекурсивный поиск
//	_ = filepath.Walk(opts.Path, func(path string, info os.FileInfo, err error) error {
//		if err != nil {
//			// Handle the error and continue walking
//			//fmt.Println(err)
//			return nil
//		}
//		if isMatch(opts.Pattern, info.Name()) && !info.IsDir() {
//			resultFiles = append(resultFiles, path)
//			//fmt.Println(path)
//		}
//		return nil
//	})
//	return resultFiles, nil
//
//}
