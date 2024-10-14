package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
)

func framedWebmToGif(input []byte, frame []byte, width int) ([]byte, error) {
	inPath := "/tmp/input.webm"
	framePath := "/tmp/frame.webm"
	outPath := "/tmp/output.gif"
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
	if width > 300 {
		width = 300
	}
	scale := getScale(width)
	cmd := exec.Command("ffmpeg",
		"-codec:v", "libvpx-vp9",
		"-y",
		"-i", inPath,
		"-i", framePath,
		"-filter_complex", "overlay=0:0,"+scale+", split=2[a][b];[b]palettegen[p];[a][p]paletteuse",
		"-loop", "0",
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
