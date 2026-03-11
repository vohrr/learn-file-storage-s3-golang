package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
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
	fmt.Printf("Content-Type: %s", imageType)

	imageDataSlice, err := io.ReadAll(imageData)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse thumbnail data into []byte", err)
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

	newUrl := fmt.Sprintf("data:%s;base64,%s", imageType, base64.StdEncoding.EncodeToString(imageDataSlice))
	videoMetadata.UpdatedAt = time.Now()
	videoMetadata.ThumbnailURL = &newUrl
	err = cfg.db.UpdateVideo(videoMetadata)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't update thumbnail metadata", err)
		return
	}
	respondWithJSON(w, http.StatusOK, videoMetadata)
}
