package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
)

func scaleWebm(input []byte, width int) ([]byte, error) {
	inPath := "/tmp/input.webm"
	outPath := "/tmp/output.webm"
	inFile, err := os.Create(inPath)
	if err != nil {
		return nil, err
	}
	defer inFile.Close()
	defer os.Remove(inPath)
	inFile.Write(input)
	outFile, err := os.Create(outPath)
	if err != nil {
		return nil, err
	}
	defer outFile.Close()
	defer os.Remove(outPath)
	scale := getScale(width)
	cmd := exec.Command("ffmpeg",
		"-y",
		"-codec:v", "libvpx-vp9",
		"-i", inPath,
		"-vf", scale,
		"-b:v", "1M",
		"-deadline", "good",
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
