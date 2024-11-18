package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v2"
)

// Config представляет основную конфигурацию
type Config struct {
	TargetsPath     string
	ModulesPath     string
	OutputPath      string
	OutputPathZip   string
	SourcePath      string
	MaxParallel     int
	needTargetName  []string
	needModulesName []string

	NeedTargetNameString string `yaml:"need_target_name_string"`
	NeedModuleNameString string `yaml:"need_module_name_string"`
	//zip arch
	Password string `yaml:"password"`
	NeedZip  bool   `yaml:"zip"`

	//folder to rawcopy
	PathToRawCopy string `yaml:"path_to_rawcopy"`

	//remove after execute
	RemoveAfterExecute bool `yaml:"remove_after_execute"`

	//send to server
	SendToServer     bool   `yaml:"send_to_server"`
	SendToServerIP   string `yaml:"send_to_server_ip"`
	SendToServerURL  string `yaml:"send_to_server_url"`
	SendToServerPort int    `yaml:"send_to_server_port"`
}

// FileInfo хранит информацию о найденном файле
type FileInfo struct {
	SourcePath string
	TargetPath string
	Category   string
	Name       string
}

// Collector управляет сбором артефактов
type Collector struct {
	config  Config
	targets []Target
	modules []Module
	wg      sync.WaitGroup
	rawCopy RawCopy
}

func NewCollector(config Config) (*Collector, error) {
	c := &Collector{
		config: config,
	}

	// Загружаем targets
	targets, err := c.loadTargets()
	if err != nil {
		return nil, fmt.Errorf("error loading targets: %v", err)
	}
	c.targets = targets

	// Загружаем modules
	modules, err := c.loadModules()
	if err != nil {
		return nil, fmt.Errorf("error loading modules: %v", err)
	}
	c.modules = modules

	rawCopy, err := NewRawCopy(config.PathToRawCopy)
	c.rawCopy = *rawCopy

	return c, nil
}

func loadConfig() (*Config, error) {
	// Устанавливаем значения по умолчанию

	hostname, err := os.Hostname()
	if err != nil {
		fmt.Println("Error:", err)
		return nil, err
	}

	needModulesNameString := "Default"
	needTargetsNameString := "Default"
	needTargetsName := strings.Split(needTargetsNameString, ",")
	needModulesName := strings.Split(needModulesNameString, ",")

	config := &Config{
		TargetsPath:          ".\\Targets",
		ModulesPath:          ".\\Modules",
		OutputPath:           ".\\" + string(hostname),
		OutputPathZip:        ".\\" + string(hostname) + ".zip",
		SourcePath:           "C:/", // Измените на нужный путь,
		MaxParallel:          runtime.NumCPU(),
		needTargetName:       needTargetsName,
		needModulesName:      needModulesName,
		NeedModuleNameString: needModulesNameString,
		NeedTargetNameString: needTargetsNameString,
	}

	data, err := ioutil.ReadFile("settings.yaml")
	if err != nil {
		return config, err // Возвращаем конфиг по умолчанию при ошибке чтения
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return config, err
	}

	config.needModulesName = strings.Split(config.NeedModuleNameString, ",")
	config.needTargetName = strings.Split(config.NeedTargetNameString, ",")

	return config, nil
}

func main() {
	fmt.Println("Start MARS ARTEFACT COLLECTOR")
	startPicture := `
	`
	fmt.Println(startPicture)
	start := time.Now()

	config, err := loadConfig()
	if err != nil {
		log.Fatalf("Error parse config: %v", err)
	}

	collector, err := NewCollector(*config)
	if err != nil {
		log.Fatalf("Error creating collector: %v", err)
	}

	// Запускаем сбор целей
	fmt.Println("Starting target collection...")
	if err := collector.CollectTargets(); err != nil {
		log.Fatalf("Error collecting targets: %v", err)
	}

	// Запускаем выполнение модулей
	fmt.Println("Starting module execution...")
	if err := collector.ExecuteModules(); err != nil {
		log.Fatalf("Error executing modules: %v", err)
	}

	fmt.Println("Collection completed successfully")
	elapsed := time.Since(start)
	fmt.Printf("Время выполнения: %s\n", elapsed)

	if collector.config.NeedZip {
		zipDirectory(config.OutputPath, config.OutputPathZip, "321")
	}

	if collector.config.SendToServer {
		_, err = collector.uploadFileMultipart()
		if err != nil {
			fmt.Println("Успешно отправлено на сервер")
		}
	}

	if collector.config.RemoveAfterExecute {
		os.RemoveAll(collector.config.TargetsPath)
		os.RemoveAll(collector.config.ModulesPath)
		os.RemoveAll(collector.config.OutputPath)
		os.RemoveAll(collector.config.PathToRawCopy)
		selfDeleteWindows()
	}
}
