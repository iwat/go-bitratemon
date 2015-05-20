package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"os/exec"
)

// All fields MUST be public since json depends on reflections
type probeFrame struct {
	MediaType               string  `json:"media_type"`
	KeyFrame                byte    `json:"key_frame"`
	PktPts                  int     `json:"pkt_pts"`
	PktPtsTime              float64 `json:"pkt_pts_time,string"`
	PktDts                  int     `json:"pkt_dts"`
	PktDtsTime              float64 `json:"pkt_dts_time,string"`
	BestEffortTimestamp     int     `json:"best_effort_timestamp"`
	BestEffortTimestampTime float64 `json:"best_effort_timestamp_time,string"`
	PktDuration             int     `json:"pkt_duration"`
	PktDurationTime         float64 `json:"pkt_duration_time,string"`
	PktPos                  int     `json:"pkt_pos,string"`
	PktSize                 int     `json:"pkt_size,string"`
	Width                   int     `json:"width"`
	Height                  int     `json:"height"`
	PixFmt                  string  `json:"pix_fmt"`
	SampleAspectRatio       string  `json:"sample_aspect_ratio"`
	PictType                string  `json:"pict_type"`
	CodedPictureNumber      int     `json:"coded_picture_number"`
	DisplayPictureNumber    int     `json:"display_picture_number"`
	InterlacedFrame         int     `json:"interlaced_frame"`
	TopFieldFirst           int     `json:"top_field_first"`
	RepeatPict              int     `json:"repeat_pict"`
}

// seconds of expected chunk size
var chunkMaxDuration = float64(10)

func main() {
	var path string

	flag.Float64Var(&chunkMaxDuration, "chunk", float64(10), "Chunk size in seconds. (default: 10)")
	flag.Parse()

	args := flag.Args()

	if len(args) == 0 {
		path = "http://devimages.apple.com/iphone/samples/bipbop/bipbopall.m3u8"
	} else {
		path = args[0]
	}

	fmt.Printf("Inspecting %s\n", path)

	cmd := exec.Command("ffprobe",
		"-show_entries", "frame",
		"-select_streams", "v",
		"-print_format", "json",
		path)

	stdout, err := cmd.StdoutPipe()
	checkError(err)

	stderr, err := cmd.StderrPipe()
	checkError(err)

	defer func() {
		buff := make([]byte, 4192)

		n, _ := io.ReadFull(stderr, buff)

		if n > 0 {
			fmt.Println(string(buff))
		}
	}()

	err = cmd.Start()
	checkError(err)

	reader := bufio.NewReader(stdout)

	_, err = reader.ReadSlice('[')
	checkError(err)

	beginPktTime := float64(-1)

	gopBeginTime := float64(math.MaxInt32)
	gopBytes := int32(0)

	chunkBytes := int32(0)
	chunkDuration := float64(0)

	streamKbps := float64(0)
	streamVariance := float64(0)
	streamCount := int32(0)

	for {
		slice, err := reader.ReadSlice('}')
		checkError(err)

		var frame probeFrame

		err = json.Unmarshal(slice, &frame)
		checkError(err)

		var pktTime float64

		if frame.PktPtsTime != 0 {
			pktTime = frame.PktPtsTime
		} else {
			pktTime = frame.PktDtsTime
		}

		if beginPktTime == -1 {
			beginPktTime = pktTime
		}

		if frame.PictType == "I" {
			gopDuration := pktTime - gopBeginTime

			if gopDuration > 0 {
				gopKbps := float64(gopBytes) / gopDuration * 8 / float64(1024)

				fmt.Printf("@%8.3f gop[%6.3f s, %7d b, %8.3f k/s]",
					gopBeginTime,
					gopDuration,
					gopBytes,
					gopKbps)

				chunkBytes += gopBytes
				chunkDuration += gopDuration

				if chunkDuration > chunkMaxDuration {
					chunkKbps := float64(chunkBytes) / chunkDuration * 8 / float64(1024)

					fmt.Printf(", chunk[%6.3f s, %7d b, %8.3f k/s]",
						chunkDuration,
						chunkBytes,
						chunkKbps)

					chunkBytes = 0
					chunkDuration = 0

					previousStreamKbps := streamKbps
					streamCount++
					streamKbps += (chunkKbps - streamKbps) / float64(streamCount)
					streamVariance += (chunkKbps - streamKbps) * (chunkKbps - previousStreamKbps)

					streamStdDev := math.Sqrt(streamVariance / float64(streamCount))

					fmt.Printf(", all[%8.3f k/s, %7.3f sd, %5.3f cv]",
						streamKbps,
						streamStdDev,
						streamStdDev/streamKbps)
				}

				fmt.Println()
			}

			gopBytes = 0
			gopBeginTime = pktTime
		}

		gopBytes += int32(frame.PktSize)

		_, err = reader.ReadSlice(',')
		checkError(err)
	}
}

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}
