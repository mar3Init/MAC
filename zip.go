package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/yeka/zip"
)

func zipDirectory(source, target, password string) error {
	// Создаем zip файл
	zipfile, err := os.Create(target)
	if err != nil {
		return err
	}
	defer zipfile.Close()

	// Создаем новый архив
	archive := zip.NewWriter(zipfile)
	defer archive.Close()

	// Проходим по всем файлам и директориям
	err = filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Получаем относительный путь
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// Устанавливаем относительный путь для файла в архиве
		relPath, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}

		// Преобразуем пути в формат ZIP (используем прямые слэши)
		header.Name = filepath.ToSlash(relPath)

		// Устанавливаем метод сжатия
		header.Method = zip.Deflate

		if info.IsDir() {
			header.Name += "/"
			// Создаем директорию в архиве
			_, err := archive.CreateHeader(header)
			return err
		}

		// Устанавливаем время модификации файла
		header.SetModTime(info.ModTime())

		var writer io.Writer
		if password == "" {
			// Создаем обычный файл без пароля
			writer, err = archive.CreateHeader(header)
		} else {
			// Создаем зашифрованный файл с улучшенными настройками
			header.Flags |= 0x1 // Установка флага шифрования
			writer, err = archive.Encrypt(header.Name, password, zip.StandardEncryption)
		}
		if err != nil {
			return err
		}

		// Открываем исходный файл
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		// Буферизированное копирование для улучшения производительности
		buf := make([]byte, 32*1024) // 32KB буфер
		_, err = io.CopyBuffer(writer, file, buf)
		return err
	})

	return err
}

func selfDeleteWindows() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe_other := exe
	if strings.Contains(exe, "64") {
		exe_other = strings.Replace(exe, "64", "32", 1)
	} else {
		exe_other = strings.Replace(exe, "32", "64", 1)
	}

	// Создаем временный bat-файл
	batPath := filepath.Join(os.TempDir(), "delete.bat")
	batContent := fmt.Sprintf(`
ping 127.0.0.1 -n 2 > nul
del "%s"
del "%s"
del "%s"
    `, exe, exe_other, batPath)

	if err := ioutil.WriteFile(batPath, []byte(batContent), 0666); err != nil {
		return err
	}

	// Запускаем bat-файл
	cmd := exec.Command("cmd", "/C", "start", "/b", batPath)
	return cmd.Start()
}
