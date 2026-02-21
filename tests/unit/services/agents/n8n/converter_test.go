package n8n_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/services/agents/n8n"
)

func TestFileConverter_SupportsFiles(t *testing.T) {
	c := n8n.NewFileConverter()
	require.True(t, c.SupportsFiles())
}

func TestFileConverter_ConvertFiles_NoFiles(t *testing.T) {
	c := n8n.NewFileConverter()
	result, err := c.ConvertFiles("hello", nil)
	require.NoError(t, err)

	req, ok := result.(*n8n.ChatRequest)
	require.True(t, ok)
	require.Equal(t, "hello", req.ChatInput)
}

func TestFileConverter_ConvertFiles_WithFiles(t *testing.T) {
	c := n8n.NewFileConverter()
	files := []n8n.FileInput{
		{Type: "file", FileData: "base64data", Filename: "test.pdf", MimeType: "application/pdf"},
	}
	result, err := c.ConvertFiles("check this", files)
	require.NoError(t, err)

	req, ok := result.(*n8n.ChatRequestWithFiles)
	require.True(t, ok)
	require.Equal(t, "check this", req.ChatInput)
	require.Len(t, req.Files, 1)
	require.Equal(t, "base64data", req.Files[0].Data)
}

func TestFileConverter_ConvertFiles_ImageWithURL(t *testing.T) {
	c := n8n.NewFileConverter()
	files := []n8n.FileInput{
		{Type: "image", ImageURL: "https://example.com/img.png", FileData: "base64"},
	}
	result, err := c.ConvertFiles("look", files)
	require.NoError(t, err)

	req, ok := result.(*n8n.ChatRequestWithFiles)
	require.True(t, ok)
	require.Equal(t, "https://example.com/img.png", req.Files[0].Data)
}

func TestFileConverter_ConvertFiles_ImageNoURL(t *testing.T) {
	c := n8n.NewFileConverter()
	files := []n8n.FileInput{
		{Type: "image", FileData: "base64data"},
	}
	result, err := c.ConvertFiles("look", files)
	require.NoError(t, err)

	req, ok := result.(*n8n.ChatRequestWithFiles)
	require.True(t, ok)
	require.Equal(t, "base64data", req.Files[0].Data)
}
