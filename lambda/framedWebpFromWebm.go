package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
)

func framedWebpFromWebm(input []byte, frame []byte, width int) ([]byte, error) {
	inPath := "/tmp/input.webm"
	framePath := "/tmp/frame.webp"
	outPath := "/tmp/output.webp"
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
		"-c:v", "libvpx-vp9",
		"-i", inPath,
		"-c:v", "libwebp",
		"-i", framePath,
		"-c:a", "copy",
		"-f:v", "1",
		"-filter_complex", "[0:v][1:v] overlay=0:0:enable='between(t,0,20)',"+scale,
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
