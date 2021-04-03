// https://www.learningcontainer.com/download/sample-mp4-files-for-download/?wpdmdl=2556&refresh=605fad01281a71616882945
package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
)

type DownloadFile struct {
	url           string
	targetPath    string
	totalSections int
}

func main() {
	fmt.Println("Downloader CLI")
	var d DownloadFile
	fmt.Print("Enter the download url: ")
	// d := DownloadFile{
	// url: "https://www.learningcontainer.com/download/sample-mp4-files-for-download/?wpdmdl=2556&refresh=605fad01281a71616882945",
	// url: "https://www.learningcontainer.com/download/sample-mp4-file/?wpdmdl=2518&refresh=605f89d08b4311616873936",
	// url: "https://www.learningcontainer.com/download/sample-mp4-video-file/?wpdmdl=2516&refresh=605f89d08ecf01616873936",
	// url: "https://www.learningcontainer.com/download/sample-video-file-for-testing/?wpdmdl=2514&refresh=605f89d0928e71616873936",
	// targetPath:    "sample-video.mp4",
	// totalSections: 10,
	// }
	fmt.Scanln(&d.url)
	fmt.Print("Enter downloaded file name: ")
	fmt.Scanln(&d.targetPath)
	d.totalSections = 10

	err := d.download()
	if err != nil {
		log.Fatalf("Error during download: %s\n", err)
	}
	fmt.Println("Download Complete. Enjoy!")
}

// download method
func (d DownloadFile) download() error {
	fmt.Println("Making Connection...")
	r, err := d.makeNewRequest("HEAD")
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return err
	}
	if resp.StatusCode > 299 {
		return errors.New(fmt.Sprint("Unable to process, response status code is " + resp.Status))
	}
	fileSize := resp.ContentLength
	if fileSize == -1 {
		return errors.New(fmt.Sprint("Undetermined content length"))
	}
	fmt.Println("File download size is " + fmt.Sprint(fileSize) + " bytes.")
	segmentSize := fileSize / int64(d.totalSections)
	if segmentSize > 5000000 {
		segmentSize = 5000000
		d.totalSections = int(fileSize) / int(segmentSize)
	} else if fileSize < 5000000 {
		segmentSize = fileSize
		d.totalSections = 1
	}
	var segments = make([][2]int, d.totalSections)
	for i := range segments {
		if i == 0 {
			segments[i][0] = 0
		} else {
			segments[i][0] = segments[i-1][1] + 1
		}
		if i < d.totalSections-1 {
			segments[i][1] = segments[i][0] + int(segmentSize)
		} else {
			segments[i][1] = int(fileSize) - 1
		}
	}

	var wg sync.WaitGroup
	for i, s := range segments {
		// concurrent segments download
		wg.Add(1)

		// local scope i
		i := i
		s := s

		go func() {
			defer wg.Done()
			err = d.downloadSegment(i, s)
			if err != nil {
				panic(errors.New(fmt.Sprintf("Download segment %d error %v", i, err)))
			}
		}()
	}
	wg.Wait()

	err = d.finishTargetFile(segments)
	if err != nil {
		return err
	}

	return nil
}

// request method
func (d DownloadFile) makeNewRequest(method string) (*http.Request, error) {
	r, err := http.NewRequest(method, d.url, nil)
	if err != nil {
		return r, err
	}
	r.Header.Set("User-Agent", "Downloader CLI v001")
	return r, nil
}

// download segment method
func (d DownloadFile) downloadSegment(i int, segment [2]int) error {
	r, err := d.makeNewRequest("GET")
	if err != nil {
		return err
	}
	r.Header.Set("Range", fmt.Sprintf("bytes=%v-%v", segment[0], segment[1]))
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return err
	}
	fmt.Println("Downloaded segment " + fmt.Sprint(i))
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	// temp file method 1
	err = ioutil.WriteFile(fmt.Sprintf("temp_%v.tmp", i), b, os.ModePerm)

	return nil
}

// target file complete creation method
func (d DownloadFile) finishTargetFile(segments [][2]int) error {
	f, err := os.OpenFile(d.targetPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModeAppend)
	if err != nil {
		return err
	}
	defer f.Close()
	for i := range segments {
		b, err := ioutil.ReadFile(fmt.Sprintf("temp_%v.tmp", i))
		if err != nil {
			// f.Close()
			return err
		}
		_, err = f.Write(b)
		if err != nil {
			// f.Close()
			return err
		}
		err = os.Remove(fmt.Sprintf("temp_%v.tmp", i))
		if err != nil {
			// f.Close()
			return err
		}
	}
	return nil
}
