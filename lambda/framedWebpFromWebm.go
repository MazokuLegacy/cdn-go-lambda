package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
)

func framedWebpFromWebm(input []byte, frame []byte, width int) ([]byte, error) {
	inPath := "/tmp/input.webm"
	framePath := "/tmp/frame.webm"
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
		"-i", framePath,
		"-c:a", "copy",
		"-filter_complex", "[0:v]select=eq(n\\,0)[bg];[bg][1:v]overlay=0:0,"+scale,
		"-frames:v", "1",
		"-y",
		outPath)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	err = cmd.Start()
	if err != nil {
		fmt.Println("Error starting command:", err)
		return nil, err
	}

	go func(reader io.ReadCloser) {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			log.Println(scanner.Text()) // Logs each line of output as it happens
		}
	}(stdout)

	// Function to read and log errors in real-time
	go func(reader io.ReadCloser) {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			log.Println(scanner.Text()) // Logs each line of errors as they happen
		}
	}(stderr)

	err = cmd.Wait()
	if err != nil {
		fmt.Println("Error waiting for command:", err)
		return nil, err
	}
	log.Println("completed")
	output, err := io.ReadAll(outFile)
	return output, nil
}
