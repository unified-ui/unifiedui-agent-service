// Package n8n provides N8N-specific agent client implementations.
package n8n

// FileInput represents a unified file attachment for N8N conversion.
type FileInput struct {
	Type     string
	ImageURL string
	FileData string
	Filename string
	MimeType string
	Detail   string
}

// FileConverter converts unified FileInput to N8N format.
type FileConverter struct{}

// NewFileConverter creates a new N8N FileConverter.
func NewFileConverter() *FileConverter {
	return &FileConverter{}
}

// SupportsFiles returns true as N8N supports files via extended webhook body.
func (c *FileConverter) SupportsFiles() bool {
	return true
}

// ConvertFiles converts FileInput slice to N8N chat request with files.
func (c *FileConverter) ConvertFiles(message string, files []FileInput) (interface{}, error) {
	if len(files) == 0 {
		return &ChatRequest{
			ChatInput: message,
		}, nil
	}

	n8nFiles := make([]FileAttachment, 0, len(files))
	for _, file := range files {
		data := file.FileData
		if file.Type == "image" && file.ImageURL != "" {
			data = file.ImageURL
		}

		n8nFiles = append(n8nFiles, FileAttachment{
			Type:     file.Type,
			Data:     data,
			Filename: file.Filename,
			MimeType: file.MimeType,
		})
	}

	return &ChatRequestWithFiles{
		ChatInput: message,
		Files:     n8nFiles,
	}, nil
}
