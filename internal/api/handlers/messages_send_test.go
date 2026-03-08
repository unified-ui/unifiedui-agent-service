package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/unifiedui/agent-service/internal/domain/models"
)

func TestConvertFilesToAttachmentMetadata_Empty(t *testing.T) {
	result := convertFilesToAttachmentMetadata(nil)
	assert.Nil(t, result)

	result = convertFilesToAttachmentMetadata([]FileAttachment{})
	assert.Nil(t, result)
}

func TestConvertFilesToAttachmentMetadata_WithFileData(t *testing.T) {
	base64Data := "SGVsbG8gV29ybGQ=" // "Hello World" in base64 (11 bytes)
	files := []FileAttachment{
		{
			Type:     "file",
			Filename: "document.pdf",
			MimeType: "application/pdf",
			FileData: base64Data,
		},
	}

	result := convertFilesToAttachmentMetadata(files)

	assert.Len(t, result, 1)
	assert.Equal(t, "document.pdf", result[0].FileName)
	assert.Equal(t, "application/pdf", result[0].FileType)
	assert.Equal(t, "file", result[0].FileCategory)
	assert.Equal(t, int64(len(base64Data)*3/4), result[0].FileSize)
}

func TestConvertFilesToAttachmentMetadata_WithImageURL(t *testing.T) {
	base64Data := "iVBORw0KGgo="
	imageURL := "data:image/png;base64," + base64Data
	files := []FileAttachment{
		{
			Type:     "image",
			Filename: "screenshot.png",
			MimeType: "image/png",
			ImageURL: imageURL,
		},
	}

	result := convertFilesToAttachmentMetadata(files)

	assert.Len(t, result, 1)
	assert.Equal(t, "screenshot.png", result[0].FileName)
	assert.Equal(t, "image/png", result[0].FileType)
	assert.Equal(t, "image", result[0].FileCategory)
	expectedSize := int64(len(base64Data) * 3 / 4)
	assert.Equal(t, expectedSize, result[0].FileSize)
}

func TestConvertFilesToAttachmentMetadata_Multiple(t *testing.T) {
	files := []FileAttachment{
		{
			Type:     "image",
			Filename: "photo.jpg",
			MimeType: "image/jpeg",
			ImageURL: "data:image/jpeg;base64,/9j/4A==",
		},
		{
			Type:     "file",
			Filename: "report.xlsx",
			MimeType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			FileData: "UEsDBBQ=",
		},
		{
			Type:     "audio",
			Filename: "voice.mp3",
			MimeType: "audio/mpeg",
			FileData: "SUQz",
		},
	}

	result := convertFilesToAttachmentMetadata(files)

	assert.Len(t, result, 3)

	assert.Equal(t, "photo.jpg", result[0].FileName)
	assert.Equal(t, "image", result[0].FileCategory)

	assert.Equal(t, "report.xlsx", result[1].FileName)
	assert.Equal(t, "file", result[1].FileCategory)

	assert.Equal(t, "voice.mp3", result[2].FileName)
	assert.Equal(t, "audio", result[2].FileCategory)
}

func TestConvertFilesToAttachmentMetadata_NoData(t *testing.T) {
	files := []FileAttachment{
		{
			Type:     "file",
			Filename: "empty.txt",
			MimeType: "text/plain",
		},
	}

	result := convertFilesToAttachmentMetadata(files)

	assert.Len(t, result, 1)
	assert.Equal(t, "empty.txt", result[0].FileName)
	assert.Equal(t, int64(0), result[0].FileSize)
}

func TestConvertFilesToAttachmentMetadata_ReturnsCorrectType(t *testing.T) {
	files := []FileAttachment{
		{
			Type:     "image",
			Filename: "test.png",
			MimeType: "image/png",
			FileData: "abc",
		},
	}

	result := convertFilesToAttachmentMetadata(files)

	assert.IsType(t, []models.AttachmentMetadata{}, result)
}
