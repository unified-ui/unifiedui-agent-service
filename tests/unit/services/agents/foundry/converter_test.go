package foundry_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unifiedui/agent-service/internal/services/agents/foundry"
)

func TestFoundryFileConverter_SupportsFiles(t *testing.T) {
	c := foundry.NewFileConverter()
	require.True(t, c.SupportsFiles())
}

func TestFoundryFileConverter_ConvertFiles_NoFiles(t *testing.T) {
	c := foundry.NewFileConverter()
	result, err := c.ConvertFiles("hello", nil)
	require.NoError(t, err)
	require.Equal(t, "hello", result)
}

func TestFoundryFileConverter_ConvertFiles_Image(t *testing.T) {
	c := foundry.NewFileConverter()
	files := []foundry.FileInput{
		{Type: "image", ImageURL: "https://example.com/img.png", Detail: "auto"},
	}
	result, err := c.ConvertFiles("describe", files)
	require.NoError(t, err)

	msgs, ok := result.([]foundry.InputMessage)
	require.True(t, ok)
	require.Len(t, msgs, 1)
	require.Len(t, msgs[0].Content, 2)
	require.Equal(t, foundry.InputTypeText, msgs[0].Content[0].Type)
	require.Equal(t, foundry.InputTypeImage, msgs[0].Content[1].Type)
	require.Equal(t, "https://example.com/img.png", msgs[0].Content[1].ImageURL)
}

func TestFoundryFileConverter_ConvertFiles_FileNoDataPrefix(t *testing.T) {
	c := foundry.NewFileConverter()
	files := []foundry.FileInput{
		{Type: "file", FileData: "rawbase64", Filename: "test.txt", MimeType: "text/plain"},
	}
	result, err := c.ConvertFiles("read", files)
	require.NoError(t, err)

	msgs, ok := result.([]foundry.InputMessage)
	require.True(t, ok)
	require.Equal(t, foundry.InputTypeFile, msgs[0].Content[1].Type)
	require.Equal(t, "data:text/plain;base64,rawbase64", msgs[0].Content[1].FileData)
}

func TestFoundryFileConverter_ConvertFiles_FileWithDataPrefix(t *testing.T) {
	c := foundry.NewFileConverter()
	files := []foundry.FileInput{
		{Type: "file", FileData: "data:text/plain;base64,abc", Filename: "test.txt", MimeType: "text/plain"},
	}
	result, err := c.ConvertFiles("read", files)
	require.NoError(t, err)

	msgs, ok := result.([]foundry.InputMessage)
	require.True(t, ok)
	require.Equal(t, "data:text/plain;base64,abc", msgs[0].Content[1].FileData)
}

func TestFoundryFileConverter_ConvertFiles_AudioWithMime(t *testing.T) {
	c := foundry.NewFileConverter()
	files := []foundry.FileInput{
		{Type: "audio", FileData: "audiodata", MimeType: "audio/wav"},
	}
	result, err := c.ConvertFiles("transcribe", files)
	require.NoError(t, err)

	msgs, ok := result.([]foundry.InputMessage)
	require.True(t, ok)
	require.Equal(t, foundry.InputTypeAudio, msgs[0].Content[1].Type)
	require.Equal(t, "audiodata", msgs[0].Content[1].Data)
	require.Equal(t, "wav", msgs[0].Content[1].Format)
}

func TestFoundryFileConverter_ConvertFiles_AudioNoMime(t *testing.T) {
	c := foundry.NewFileConverter()
	files := []foundry.FileInput{
		{Type: "audio", FileData: "audiodata"},
	}
	result, err := c.ConvertFiles("transcribe", files)
	require.NoError(t, err)

	msgs, ok := result.([]foundry.InputMessage)
	require.True(t, ok)
	require.Equal(t, "mp3", msgs[0].Content[1].Format)
}

func TestFoundryFileConverter_ConvertFiles_Mixed(t *testing.T) {
	c := foundry.NewFileConverter()
	files := []foundry.FileInput{
		{Type: "image", ImageURL: "https://example.com/img.png"},
		{Type: "file", FileData: "data", Filename: "f.txt", MimeType: "text/plain"},
		{Type: "audio", FileData: "audio", MimeType: "audio/mp3"},
	}
	result, err := c.ConvertFiles("analyze all", files)
	require.NoError(t, err)

	msgs, ok := result.([]foundry.InputMessage)
	require.True(t, ok)
	require.Len(t, msgs[0].Content, 4)
}
