package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// RAW COPY
// RAW COPY
// RAW COPY
// RAW COPY
// RAW COPY
// RAW COPY
// RAW COPY
// RAW COPY
// RAW COPY
// RawCopy представляет собой структуру для работы с утилитой RawCopy
type RawCopy struct {
	execPath string // путь к RawCopy.exe
}

// NewRawCopy создает новый экземпляр RawCopy
func NewRawCopy(pathToExe string) (*RawCopy, error) {
	absPath, err := filepath.Abs(pathToExe)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения абсолютного пути: %v", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("RawCopy.exe не найден по пути: %s", absPath)
	}

	return &RawCopy{execPath: absPath}, nil
}

// CopyFile копирует файл с помощью RawCopy
func (rc *RawCopy) CopyFile(srcPath string, dstPath string) error {
	// Проверяем существование исходного файла
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return fmt.Errorf("исходный файл не существует: %s", srcPath)
	}

	// Создаём директорию назначения, если она не существует
	dstDir := filepath.Dir(dstPath)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("ошибка создания директории назначения: %v", err)
	}

	// Подготавливаем команду
	// RawCopy использует формат: RawCopy.exe /FileNameSrc:source /FileNameDst:destination
	if strings.Contains(srcPath, "$MFT") {
		srcPath = "C:0"
	}
	if strings.Contains(srcPath, "$J") {
		srcPath = "c:\\$Extend\\$UsnJrnl"
	}
	if strings.Contains(dstDir, "*") {
		dstDir = strings.Replace(dstDir, "*", "", 10)
	}
	cmd := exec.Command(rc.execPath,
		"/FileNamePath:"+srcPath,
		"/OutputPath:"+dstPath)

	// Получаем вывод команды
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ошибка выполнения RawCopy: %v\nВывод: %s", err, string(output))
	}

	return nil
}

func copyFile(src, dst string) error {
	sourceFileNameSlice := strings.Split(src, "\\")
	var sourceFileName = sourceFileNameSlice[len(sourceFileNameSlice)-1]
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst + "\\" + string(sourceFileName))
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	return destFile.Sync()
}

func (rc *RawCopy) fullCopy(srcPath string, dstPath string) error {
	err := createDirRecursively(dstPath)
	if err != nil {
		fmt.Printf("Ошибка: %v\n", err)
		return err
	}

	err = copyFile(srcPath, dstPath)
	if err != nil {
		//fmt.Println("Error copying file:", srcPath)
		//fmt.Println("Try RawCopy")
		err2 := rc.CopyFile(srcPath, dstPath)
		if err2 != nil {
			fmt.Println("Error Raw Copy:", srcPath)
		} else {
			//	fmt.Printf("File %s scopied with Raw\n", srcPath)
		}
		return err2
	} else {
		//fmt.Println("File copied successfully")
	}
	return err
}

func createDirRecursively(path string) error {
	// os.MkdirAll создает все родительские директории, если они не существуют
	err := os.MkdirAll(path, 0755) // 0755 это права доступа: rwxr-xr-x
	if err != nil {
		return fmt.Errorf("ошибка при создании директории: %v", err)
	}
	return nil
}

//END RAW COPY
//END RAW COPY
//END RAW COPY
//END RAW COPY
//END RAW COPY
//END RAW COPY
//END RAW COPY
//END RAW COPY
//END RAW COPY
