package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	err = r.ParseMultipartForm(10 << 20)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse multi-part form", err)
		return
	}
	imageData, imageHeader, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse formFile", err)
		return
	}

	imageType := imageHeader.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(imageType)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Failed to parse mime type", err)
		return
	}
	if mediaType != "image/jpeg" && mediaType != "image/png" {
		respondWithError(w, http.StatusBadRequest, "Invalid file type", err)
		return
	}
	videoMetadata, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Video not found", err)
		return
	}
	if videoMetadata.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Not authorized for this action", err)
		return
	}

	//create file on disk
	var fileExtension string
	imageTypeSlice := strings.Split(imageType, "/")
	if len(imageTypeSlice) == 2 {
		fileExtension = imageTypeSlice[1]
	} else {
		fileExtension = imageType

	}
	var file []byte
	_, err = rand.Read(file)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Video not found", err)
		return
	}
	fileName := fmt.Sprintf("%s.%s", base64.RawStdEncoding.EncodeToString(file), fileExtension)
	filePath := filepath.Join(cfg.assetsRoot, fileName)
	thumbnailFile, err := os.Create(filePath)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Failed to save thumbnail file", err)
		return

	}
	//write data to file
	_, err = io.Copy(thumbnailFile, imageData)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Failed to save thumbnail file", err)
		return
	}

	//update metadata with url
	thumbnailUrl := fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, fileName)
	videoMetadata.UpdatedAt = time.Now()
	videoMetadata.ThumbnailURL = &thumbnailUrl
	err = cfg.db.UpdateVideo(videoMetadata)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't update thumbnail metadata", err)
		return
	}
	respondWithJSON(w, http.StatusOK, videoMetadata)
}
