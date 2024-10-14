package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
)

func pngToFramedMp4(input []byte, frame []byte, width int) ([]byte, error) {
	inPath := "/tmp/input.png"
	framePath := "/tmp/frame.webm"
	outPath := "/tmp/output.mp4"
	inFile, err := os.Create(inPath)
	if err != nil {
		return nil, err
	}
	defer inFile.Close()
	defer os.Remove(inPath)
	inFile.Write(input)
	frameFile, err := os.Create(framePath)
	if err != nil {
		return nil, err
	}
	defer frameFile.Close()
	defer os.Remove(inPath)
	frameFile.Write(frame)
	outFile, err := os.Create(outPath)
	if err != nil {
		return nil, err
	}
	defer outFile.Close()
	defer os.Remove(outPath)
	scale := getScale(width)
	cmd := exec.Command("ffmpeg",
		"-i", inPath,
		"-i", framePath,
		"-filter_complex", "[0:v][1:v] overlay=0:0"+scale,
		"-an",
		"-c:v", "libx264",
		"-crf", "23",
		"-y",
		outPath)
	err = cmd.Start()
	if err != nil {
		fmt.Println("Error starting command:", err)
		return nil, err
	}
	err = cmd.Wait()
	if err != nil {
		fmt.Println("Error waiting for command:", err)
		return nil, err
	}
	log.Println("completed")
	output, err := io.ReadAll(outFile)
	return output, nil
}
