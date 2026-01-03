package cdl

import (
	"fmt"
	"io/ioutil"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// Parser CDL 解析器
type Parser struct {
	log *logrus.Entry
}

// NewParser 创建 CDL 解析器
func NewParser(log *logrus.Entry) *Parser {
	if log == nil {
		log = logrus.NewEntry(logrus.New())
	}
	return &Parser{
		log: log,
	}
}

// Parse 从 YAML 字符串解析 CDL
func (p *Parser) Parse(yamlContent string) (*CDLDescriptor, error) {
	p.log.Debug("Parsing CDL from YAML string")

	var descriptor CDLDescriptor
	err := yaml.Unmarshal([]byte(yamlContent), &descriptor)
	if err != nil {
		p.log.WithError(err).Error("Failed to parse YAML")
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	p.log.WithFields(logrus.Fields{
		"name":    descriptor.Consensus.Name,
		"version": descriptor.Consensus.Version,
		"type":    descriptor.Consensus.Type,
	}).Info("Successfully parsed CDL descriptor")

	return &descriptor, nil
}

// ParseFile 从文件解析 CDL
func (p *Parser) ParseFile(filePath string) (*CDLDescriptor, error) {
	p.log.WithField("file", filePath).Debug("Parsing CDL from file")

	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		p.log.WithError(err).Error("Failed to read file")
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return p.Parse(string(data))
}

// ParseAndValidate 解析并验证 CDL
func (p *Parser) ParseAndValidate(yamlContent string) (*CDLDescriptor, error) {
	descriptor, err := p.Parse(yamlContent)
	if err != nil {
		return nil, err
	}

	// 执行基本验证
	if err := descriptor.Validate(); err != nil {
		p.log.WithError(err).Error("CDL validation failed")
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	p.log.Info("CDL parsed and validated successfully")
	return descriptor, nil
}

// ParseFileAndValidate 从文件解析并验证 CDL
func (p *Parser) ParseFileAndValidate(filePath string) (*CDLDescriptor, error) {
	descriptor, err := p.ParseFile(filePath)
	if err != nil {
		return nil, err
	}

	if err := descriptor.Validate(); err != nil {
		p.log.WithError(err).Error("CDL validation failed")
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	p.log.Info("CDL parsed and validated successfully")
	return descriptor, nil
}

// Serialize 序列化 CDL 为 YAML
func (p *Parser) Serialize(descriptor *CDLDescriptor) (string, error) {
	p.log.Debug("Serializing CDL to YAML")

	data, err := yaml.Marshal(descriptor)
	if err != nil {
		p.log.WithError(err).Error("Failed to serialize CDL")
		return "", fmt.Errorf("failed to serialize CDL: %w", err)
	}

	return string(data), nil
}

// SerializeToFile 序列化 CDL 到文件
func (p *Parser) SerializeToFile(descriptor *CDLDescriptor, filePath string) error {
	p.log.WithField("file", filePath).Debug("Serializing CDL to file")

	yamlContent, err := p.Serialize(descriptor)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filePath, []byte(yamlContent), 0644)
	if err != nil {
		p.log.WithError(err).Error("Failed to write file")
		return fmt.Errorf("failed to write file: %w", err)
	}

	p.log.Info("CDL serialized to file successfully")
	return nil
}
