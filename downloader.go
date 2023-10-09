package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
)

type Downloader struct {
	Url           string
	State         int
	readFd        io.ReadCloser
	ReceivedBytes int
	ContentLength int64
}

func (downloader *Downloader) Get(r *http.Request) (*http.Response, error) {
	if downloader.State != 0 {
		err := errors.New(fmt.Sprintf("Bad satate: %d", downloader.State))
		ErrorLog.Println(err)
		return nil, err
	}
	downloader.State = 1
	client := http.Client{}
	request, err := http.NewRequest(http.MethodGet, downloader.Url, nil)
	request.Header = r.Header.Clone()
	request.Header.Del(`Authorization`)
	if err != nil {
		ErrorLog.Println(err.Error())
		return nil, err
	}
	response, err := client.Do(request)
	if err != nil {
		ErrorLog.Println(err.Error())
		return nil, err
	}
	if response.StatusCode != 200 {
		response.Body.Close()
		err := errors.New(response.Status)
		ErrorLog.Println(err)
		return nil, err
	}
	downloader.readFd = response.Body
	downloader.State = 2
	return response, nil
}

func (downloader *Downloader) Download(primaryDst, secondaryDst io.Writer) error {
	if downloader.State != 2 {
		err := errors.New(fmt.Sprintf("Bad satate: %d", downloader.State))
		ErrorLog.Println(err)
		return err
	}
	downloader.State = 3
	defer func() {
		downloader.readFd.Close()
		downloader.State = 4
	}()
	var secondaryErr error
	for {
		p := make([]byte, 1024)
		n, err := downloader.readFd.Read(p)
		if n > 0 {
			n1, err1 := primaryDst.Write(p[0:n])
			downloader.ReceivedBytes = downloader.ReceivedBytes + n1
			if err1 != nil {
				ErrorLog.Println(err.Error())
				return err1
			}
			if secondaryDst != nil && secondaryErr == nil {
				if _, err2 := secondaryDst.Write(p[0:n]); err2 != nil {
					ErrorLog.Println(err2.Error())
					secondaryErr = err2
				}
			}
		}
		if err == io.EOF {
			return nil
		} else if err != nil {
			ErrorLog.Println(err.Error())
			return err
		}
	}
}

func NewDownloader(url string) *Downloader {
	downloader := Downloader{
		Url: url,
	}
	return &downloader
}
