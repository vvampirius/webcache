package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"time"
)

const (
	CacheStateNotExist           = 0
	CacheStateDownloadInProgress = 1
	CacheStateFound              = 2
)

type Cache struct {
	filePath           string
	metaPath           string
	writeFd            io.WriteCloser
	readFd             io.ReadCloser
	Url                string    `json:"url"`
	WriteInProgressPid int       `json:"write_in_progress_pid"`
	Username           string    `json:"username"`
	ContentType        string    `json:"content-type"`
	ContentDisposition string    `json:"content-disposition"`
	Ttl                time.Time `json:"ttl"`
}

func (cache *Cache) IsMetaExist() bool {
	if _, err := os.Stat(cache.metaPath); os.IsNotExist(err) {
		return false
	}
	return true
}

func (cache *Cache) WriteInProgress() bool {
	if cache.WriteInProgressPid != 0 && cache.WriteInProgressPid == os.Getpid() {
		return true
	}
	return false
}

func (cache *Cache) State() int8 {
	if !cache.IsMetaExist() {
		return CacheStateNotExist
	} else if cache.WriteInProgress() {
		return CacheStateDownloadInProgress
	}
	return CacheStateFound
}

func (cache *Cache) Load() error {
	f, err := os.Open(cache.metaPath)
	if err != nil {
		if !os.IsNotExist(err) {
			ErrorLog.Println(err.Error())
		}
		return err
	}
	defer f.Close()
	decoder := json.NewDecoder(f)
	if err := decoder.Decode(cache); err != nil {
		ErrorLog.Println(err.Error())
	}
	return nil
}

func (cache *Cache) Save() error {
	f, err := os.OpenFile(cache.metaPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		ErrorLog.Println(err.Error())
		return err
	}
	defer f.Close()
	encoder := json.NewEncoder(f)
	if err := encoder.Encode(*cache); err != nil {
		ErrorLog.Println(err.Error())
		return err
	}
	return nil
}

func (cache *Cache) Write(p []byte) (int, error) {
	if cache.writeFd == nil {
		f, err := os.OpenFile(cache.filePath, os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return 0, err
		}
		cache.writeFd = f
	}
	return cache.writeFd.Write(p)
}

func (cache *Cache) Read(p []byte) (int, error) {
	if cache.readFd == nil {
		f, err := os.Open(cache.filePath)
		if err != nil {
			return 0, err
		}
		cache.readFd = f
	}
	return cache.readFd.Read(p)
}

func (cache *Cache) Close() error {
	if cache.writeFd != nil {
		cache.writeFd.Close()
		cache.writeFd = nil
	}
	if cache.readFd != nil {
		cache.readFd.Close()
		cache.readFd = nil
	}
	return nil
}

func (cache *Cache) Size() int64 {
	i, _ := os.Stat(cache.filePath)
	return i.Size()
}

func (cache *Cache) Remove() error {
	DebugLog.Println(`Removing`, cache)
	cache.Close()
	var err error
	if err1 := os.Remove(cache.filePath); err1 != nil {
		ErrorLog.Println(err1.Error())
		err = err1
	}
	if err2 := os.Remove(cache.metaPath); err2 != nil {
		ErrorLog.Println(err2.Error())
		err = err2
	}
	return err
}

func (cache *Cache) SetTTL(s string) error {
	d, err := time.ParseDuration(s)
	if err != nil {
		ErrorLog.Println(err.Error())
		return err
	}
	cache.Ttl = time.Now().Add(d)
	go func() {
		<-time.After(d)
		cache.Remove()
	}()
	return nil
}

func NewCache(storagePath string, url string) (*Cache, error) {
	hash := md5.Sum([]byte(url))
	urlMd5 := hex.EncodeToString(hash[:])
	cache := Cache{
		filePath: path.Join(storagePath, fmt.Sprintf("%s.file", urlMd5)),
		metaPath: path.Join(storagePath, fmt.Sprintf("%s.json", urlMd5)),
	}
	if err := cache.Load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return &cache, nil
}

func ScheduleCacheRemove(storagePath string) error {
	entries, err := os.ReadDir(storagePath)
	if err != nil {
		ErrorLog.Println(err.Error())
		return err
	}
	r := regexp.MustCompile(`\.json$`)
	for _, entry := range entries {
		fileName := entry.Name()
		if r.MatchString(fileName) {
			cache := Cache{
				metaPath: path.Join(storagePath, fileName),
				filePath: path.Join(storagePath, r.ReplaceAllString(fileName, `.file`)),
			}
			if err := cache.Load(); err != nil {
				continue
			}
			if cache.Ttl.IsZero() {
				continue
			}
			if cache.Ttl.Before(time.Now()) {
				cache.Remove()
			}
			d := cache.Ttl.Sub(time.Now())
			go func() {
				<-time.After(d)
				cache.Remove()
			}()
		}
	}
	return nil
}
