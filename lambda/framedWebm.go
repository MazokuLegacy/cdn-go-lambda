package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
)

func framedWebm(input []byte, frame []byte, width int) ([]byte, error) {
	inPath := "/tmp/input.webm"
	framePath := "/tmp/frame.webm"
	outPath := "/tmp/output.webm"
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
	log.Println("process begins")
	scale := getScale(width)
	cmd := exec.Command("ffmpeg",
		"-c:v", "libvpx-vp9",
		"-i", inPath,
		"-c:v", "libvpx-vp9",
		"-i", framePath,
		"-c:a", "copy",
		"-filter_complex", "[0:v][1:v] overlay=0:0:enable='between(t,0,20)',"+scale,
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
	// Function to read and log output in real-time
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
