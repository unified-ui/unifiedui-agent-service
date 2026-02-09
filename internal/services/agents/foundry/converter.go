// Package foundry provides Microsoft Foundry agent client implementations.
package foundry

import (
	"strings"
)

// FileInput represents a unified file attachment for Foundry conversion.
type FileInput struct {
	Type     string
	ImageURL string
	FileData string
	Filename string
	MimeType string
	Detail   string
}

// FileConverter converts unified FileInput to Foundry multimodal format.
type FileConverter struct{}

// NewFileConverter creates a new Foundry FileConverter.
func NewFileConverter() *FileConverter {
	return &FileConverter{}
}

// SupportsFiles returns true as Foundry supports multimodal input.
func (c *FileConverter) SupportsFiles() bool {
	return true
}

// ConvertFiles converts FileInput slice to Foundry multimodal content.
func (c *FileConverter) ConvertFiles(message string, files []FileInput) (interface{}, error) {
	if len(files) == 0 {
		return message, nil
	}

	content := []InputContent{
		{Type: InputTypeText, Text: message},
	}

	for _, file := range files {
		switch file.Type {
		case "image":
			content = append(content, InputContent{
				Type:     InputTypeImage,
				ImageURL: file.ImageURL,
				Detail:   file.Detail,
			})
		case "file":
			fileData := file.FileData
			if file.MimeType != "" && !strings.HasPrefix(fileData, "data:") {
				fileData = "data:" + file.MimeType + ";base64," + fileData
			}
			content = append(content, InputContent{
				Type:     InputTypeFile,
				FileData: fileData,
				Filename: file.Filename,
			})
		case "audio":
			content = append(content, InputContent{
				Type:   InputTypeAudio,
				Data:   file.FileData,
				Format: extractAudioFormat(file.MimeType),
			})
		}
	}

	return []InputMessage{
		{Role: "user", Content: content},
	}, nil
}

// extractAudioFormat extracts the audio format from MIME type.
func extractAudioFormat(mimeType string) string {
	if mimeType == "" {
		return "mp3"
	}
	parts := strings.Split(mimeType, "/")
	if len(parts) == 2 {
		return parts[1]
	}
	return "mp3"
}
