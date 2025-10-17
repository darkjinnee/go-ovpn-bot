package ovpn

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Service struct {
	scriptsPath  string
	configsPath  string
	configPrefix string
}

func New(scriptsPath, configsPath, configPrefix string) *Service {
	return &Service{
		scriptsPath:  scriptsPath,
		configsPath:  configsPath,
		configPrefix: configPrefix,
	}
}

// GenerateRandomName генерирует случайное имя для клиента
func (s *Service) GenerateRandomName() string {
	rand.Seed(time.Now().UnixNano())
	
	// Генерируем 8 случайных символов в верхнем регистре
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	randomPart := make([]byte, 8)
	
	for i := range randomPart {
		randomPart[i] = charset[rand.Intn(len(charset))]
	}
	
	return s.configPrefix + string(randomPart)
}

// CreateClient создает нового клиента OpenVPN
func (s *Service) CreateClient() (string, string, error) {
	// Генерируем случайное имя
	clientName := s.GenerateRandomName()
	
	// Создаем директорию для конфигов если она не существует
	if err := os.MkdirAll(s.configsPath, 0755); err != nil {
		return "", "", fmt.Errorf("failed to create configs directory: %w", err)
	}
	
	// Путь к скрипту add.sh
	addScript := filepath.Join(s.scriptsPath, "add.sh")
	
	// Выполняем скрипт add.sh
	cmd := exec.Command("sudo", addScript, clientName, s.configsPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", "", fmt.Errorf("failed to create client: %w, output: %s", err, string(output))
	}
	
	// Скрипт возвращает путь к созданному файлу
	configPath := strings.TrimSpace(string(output))
	
	// Проверяем что файл действительно создан
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return "", "", fmt.Errorf("config file was not created: %s", configPath)
	}
	
	return clientName, configPath, nil
}

// RemoveClient удаляет клиента OpenVPN
func (s *Service) RemoveClient(clientName, configPath string) error {
	// Путь к скрипту remove.sh
	removeScript := filepath.Join(s.scriptsPath, "remove.sh")
	
	// Выполняем скрипт remove.sh
	cmd := exec.Command("sudo", removeScript, clientName, configPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove client: %w, output: %s", err, string(output))
	}
	
	return nil
}

// ListClients возвращает список всех клиентов OpenVPN
func (s *Service) ListClients() ([]string, error) {
	// Путь к скрипту remove.sh с флагом --list
	removeScript := filepath.Join(s.scriptsPath, "remove.sh")
	
	// Выполняем скрипт remove.sh --list
	cmd := exec.Command("sudo", removeScript, "--list")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to list clients: %w, output: %s", err, string(output))
	}
	
	// Парсим вывод скрипта
	lines := strings.Split(string(output), "\n")
	var clients []string
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Пропускаем заголовки и пустые строки
		if line == "" || strings.Contains(line, "Available OpenVPN clients") || 
		   strings.Contains(line, "No clients found") {
			continue
		}
		
		// Убираем номера из начала строки (например "1) client_name")
		if idx := strings.Index(line, ") "); idx != -1 {
			line = line[idx+2:]
		}
		
		if line != "" {
			clients = append(clients, line)
		}
	}
	
	return clients, nil
}

// ReadConfigFile читает содержимое конфигурационного файла
func (s *Service) ReadConfigFile(configPath string) ([]byte, error) {
	return os.ReadFile(configPath)
}
