package ffmpeg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os/exec"
)

func GetVideoAspectRatio(filePath string) (string, error) {
	args := []string{"-v", "error", "-print_format", "json", "-show_streams", filePath}

	cmd := exec.Command("ffprobe", args...)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	err := cmd.Run()
	if err != nil {
		return "", err
	}

	var probeOutput struct {
		Streams []stream `json:"streams"`
	}
	err = json.Unmarshal(stdout.Bytes(), &probeOutput)
	if err != nil {
		return "", err
	}

	var streamWithWidthAndHeight stream
	for _, strm := range probeOutput.Streams {
		if strm.Width != 0 && strm.Height != 0 {
			streamWithWidthAndHeight = strm
			break
		}
	}

	if streamWithWidthAndHeight.Width == 0 || streamWithWidthAndHeight.Height == 0 {
		return "", fmt.Errorf("no video stream with width and height found")
	}

	aspectRatio := float64(streamWithWidthAndHeight.Width) / float64(streamWithWidthAndHeight.Height)

	closestRatio := "other"
	smallestDiff := math.MaxFloat64
	for _, candidate := range []struct {
		label string
		value float64
	}{
		{label: "16:9", value: 16.0 / 9.0},
		{label: "9:16", value: 9.0 / 16.0},
	} {
		diff := math.Abs(aspectRatio - candidate.value)
		if diff < smallestDiff {
			smallestDiff = diff
			closestRatio = candidate.label
		}
	}

	if smallestDiff > 0.1 {
		return "other", nil
	}

	return closestRatio, nil
}

type stream struct {
	Index              int    `json:"index"`
	CodecName          string `json:"codec_name"`
	CodecLongName      string `json:"codec_long_name"`
	Profile            string `json:"profile"`
	CodecType          string `json:"codec_type"`
	CodecTagString     string `json:"codec_tag_string"`
	CodecTag           string `json:"codec_tag"`
	Width              int    `json:"width"`
	Height             int    `json:"height"`
	CodedWidth         int    `json:"coded_width"`
	CodedHeight        int    `json:"coded_height"`
	ClosedCaptions     int    `json:"closed_captions"`
	HasBFrames         int    `json:"has_b_frames"`
	SampleAspectRatio  string `json:"sample_aspect_ratio"`
	DisplayAspectRatio string `json:"display_aspect_ratio"`
	PixFmt             string `json:"pix_fmt"`
	Level              int    `json:"level"`
	ColorRange         string `json:"color_range"`
	ColorSpace         string `json:"color_space"`
	ColorTransfer      string `json:"color_transfer"`
	ColorPrimaries     string `json:"color_primaries"`
	ChromaLocation     string `json:"chroma_location"`
	Refs               int    `json:"refs"`
	IsAvc              string `json:"is_avc"`
	NalLengthSize      string `json:"nal_length_size"`
	RFrameRate         string `json:"r_frame_rate"`
	AvgFrameRate       string `json:"avg_frame_rate"`
	TimeBase           string `json:"time_base"`
	StartPts           int    `json:"start_pts"`
	StartTime          string `json:"start_time"`
	DurationTs         int    `json:"duration_ts"`
	Duration           string `json:"duration"`
	BitRate            string `json:"bit_rate"`
	BitsPerRawSample   string `json:"bits_per_raw_sample"`
	NbFrames           string `json:"nb_frames"`
	Disposition        struct {
		Default         int `json:"default"`
		Dub             int `json:"dub"`
		Original        int `json:"original"`
		Comment         int `json:"comment"`
		Lyrics          int `json:"lyrics"`
		Karaoke         int `json:"karaoke"`
		Forced          int `json:"forced"`
		HearingImpaired int `json:"hearing_impaired"`
		VisualImpaired  int `json:"visual_impaired"`
		CleanEffects    int `json:"clean_effects"`
		AttachedPic     int `json:"attached_pic"`
		TimedThumbnails int `json:"timed_thumbnails"`
	} `json:"disposition"`
	Tags struct {
		Language    string `json:"language"`
		HandlerName string `json:"handler_name"`
		VendorID    string `json:"vendor_id"`
		Encoder     string `json:"encoder"`
		Timecode    string `json:"timecode"`
	} `json:"tags"`
}
