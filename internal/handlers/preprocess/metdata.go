package preprocess

import (
	"fmt"
	"net/http"
	"time"	
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"convo/internal/middleware"
	"convo/internal/utils"
)

type MetadataHandler struct {

}

func (h *MetadataHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
    if !ok {
        // http.Error(w, "Unauthorized", http.StatusUnauthorized)
		utils.JSON(w, http.StatusUnauthorized, utils.APIResponse{
			Success: false,
			Message: "Unauthorized",
		})
        return
    }

	// Parse the form data to get the file
	file, header,err := r.FormFile("file")
	if err != nil {
		utils.JSON(w, http.StatusBadRequest, utils.APIResponse{
			Success: false,
			Message: "Failed to parse file",
			Data: header,
		})
		return
	}
	defer file.Close()

	// Create a temporary file to store the uploaded file
	tempPath := filepath.Join(os.TempDir(), fmt.Sprintf("upload-%d-%s-%d", userID, "convo",time.Now().UnixNano()))
	tempFile, err := os.Create(tempPath)
	if err != nil {
		utils.JSON(w, http.StatusInternalServerError, utils.APIResponse{
			Success: false,
			Message: "Failed to create temporary file",
		})
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()
	io.Copy(tempFile, file)

	// Run the metadata extraction command
	cmd := exec.Command("./cpp/img_parser", tempFile.Name())
	output, err := cmd.Output()
	if err != nil {
		utils.JSON(w, http.StatusInternalServerError, utils.APIResponse{
			Success: false,
			Message: "Failed to extract metadata",
		})
		return
	}
	// Parse the output to extract metadata
	metadata := make(map[string]string)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		metadata[key] = value
	}
	// Return the metadata as JSON response
	utils.JSON(w, http.StatusOK, utils.APIResponse{
		Success: true,
		Message: "Metadata extracted successfully",
		Data:    metadata,
	})	

}