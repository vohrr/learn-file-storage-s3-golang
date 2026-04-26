package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/ffmpeg"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<30)
	// get videoID from path
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}
	//get bearer token
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}
	//authorize
	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	videoMetadata, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Video not found", err)
		return
	}

	if videoMetadata.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Not Authorized", nil)
		return
	}

	videoData, videoHeader, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not parse form data", err)
		return
	}

	defer videoData.Close()

	fileType := videoHeader.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(fileType)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Failed to parse mime type", err)
		return
	}

	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Invalid file type", err)
		return
	}
	//parse request data into temp file
	tempFile, err := os.CreateTemp("", "tubely-5000-upload.mp4")
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()
	_, err = io.Copy(tempFile, videoData)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to copy file data to temp", err)
		return
	}
	tempFile.Seek(0, io.SeekStart)

	filePath := tempFile.Name()
	aspectRatio, err := ffmpeg.GetVideoAspectRatio(filePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to inspect video", err)
		return
	}

	var prefix string
	if aspectRatio == "16:9" {
		prefix = "landscape"
	} else if aspectRatio == "9:16" {
		prefix = "portrait"
	} else {
		prefix = "other"
	}

	processingFilePath, err := ffmpeg.ProcessVideoForFastStart(filePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to convert video for fast processing", err)
		return
	}
	fastProcessFile, err := os.Open(processingFilePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to open fast processing file", err)
		return
	}

	defer os.Remove(processingFilePath)
	defer fastProcessFile.Close()

	keyHex := make([]byte, 32)
	_, err = rand.Read(keyHex)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Video not found", err)
		return
	}
	key := fmt.Sprintf("%s/%s", prefix, hex.EncodeToString(keyHex))
	//create object input and push to  s3 client
	params := s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &key,
		Body:        fastProcessFile,
		ContentType: &mediaType,
	}
	_, err = cfg.s3Client.PutObject(r.Context(), &params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to upload file", err)
		return
	}

	//update video metadata
	videoMetadata.VideoURL = MakeVideoURL(cfg, key)
	err = cfg.db.UpdateVideo(videoMetadata)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update Video URL", err)
		return
	}

}

func MakeVideoURL(cfg *apiConfig, key string) *string {
	url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, key)
	return &url
}
