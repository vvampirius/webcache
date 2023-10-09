package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

type Core struct {
	StoragePath string
	Insecure    bool
	Auth        *Auth
	Downloaders map[string]*Downloader
}

// getBasicAuthUsername Get and check password from Basic_Auth
func (core *Core) getBasicAuthUsername(r *http.Request) string {
	username, password, ok := r.BasicAuth()
	if !ok {
		return ``
	}
	if core.Auth.IsPasswordValid(username, password) {
		return username
	}
	return ``
}

func (core *Core) GetCache(url string) (*Cache, error) {
	cache, err := NewCache(core.StoragePath, url)
	if err != nil {
		return nil, err
	}
	if cache.Url != `` && cache.Url != url {
		err := errors.New(fmt.Sprintf("URL mismatch: %s != %s", cache.Url, url))
		ErrorLog.Println(err.Error())
		return nil, err
	}
	return cache, nil
}

func (core *Core) MainHttpHandler(w http.ResponseWriter, r *http.Request) {
	username := core.getBasicAuthUsername(r)
	DebugLog.Println(username, r.Method, r.RequestURI, r.Referer())
	cacheUrl := r.URL.Query().Get(`url`)
	if r.Method == http.MethodGet {
		if cacheUrl == `` {
			if username == `` && core.Insecure == false {
				w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Header().Set(`Content-Type`, `text/html; charset=utf-8`)
			fmt.Fprint(w, ` <form action="" method="get">
			  <label for="url">Enter URL:</label>&nbsp;<input type="text" id="url" name="url" autofocus required><input type="submit" value="Submit">
				</form>`)
			return
		}
		cache, err := core.GetCache(cacheUrl)
		defer cache.Close()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		cacheState := cache.State()

		// Response 404 for anonymous if not insecure mode
		if cacheState != 2 && username == `` && core.Insecure == false {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if r.URL.Query().Has(`background`) {
			if cacheState == CacheStateNotExist {
				cache.Url = cacheUrl
				cache.WriteInProgressPid = os.Getpid()
				cache.Username = username
				if ttl := r.URL.Query().Get(`ttl`); ttl != `` {
					if err := cache.SetTTL(ttl); err != nil {
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
				}
				if err := cache.Save(); err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				downloader := NewDownloader(cacheUrl)
				core.Downloaders[cacheUrl] = downloader
				go func() {
					defer func() {
						<-time.After(48 * time.Hour)
						delete(core.Downloaders, cacheUrl)
					}()
					response, err := downloader.Get(r)
					if err != nil {
						cache.Remove()
						return
					}
					cache.ContentType = response.Header.Get(`Content-Type`)
					cache.ContentDisposition = response.Header.Get(`Content-Disposition`)
					if contentLength := response.Header.Get(`Content-Length`); contentLength != `` {
						if n, err := strconv.ParseInt(response.Header.Get(`Content-Length`), 10, 64); err == nil {
							downloader.ContentLength = n
						}
					}
					if err := downloader.Download(cache, nil); err != nil {
						cache.Remove()
						return
					}
					cache.WriteInProgressPid = 0
					cache.Save()
				}()
				w.Header().Set(`Content-Type`, `application/json`)
				encoder := json.NewEncoder(w)
				encoder.Encode(*downloader)
				return
			}
			return
		}

		switch cacheState {
		case CacheStateNotExist:
			cache.Url = cacheUrl
			cache.WriteInProgressPid = os.Getpid()
			cache.Username = username
			if ttl := r.URL.Query().Get(`ttl`); ttl != `` {
				if err := cache.SetTTL(ttl); err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			}
			if err := cache.Save(); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			downloader := NewDownloader(cacheUrl)
			response, err := downloader.Get(r)
			if err != nil {
				cache.Remove()
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			cache.ContentType = response.Header.Get(`Content-Type`)
			w.Header().Set(`Content-Type`, response.Header.Get(`Content-Type`))
			cache.ContentDisposition = response.Header.Get(`Content-Disposition`)
			if cache.ContentDisposition != `` {
				w.Header().Set(`Content-Disposition`, cache.ContentDisposition)
			}
			if contentLength := response.Header.Get(`Content-Length`); contentLength != `` {
				w.Header().Set(`Content-Length`, contentLength)
			}
			w.Header().Set(`X-Cache-State`, `not found`)
			DebugLog.Println(response.Header)
			if err := downloader.Download(cache, w); err != nil {
				cache.Remove()
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			cache.WriteInProgressPid = 0
			cache.Save()
		case CacheStateDownloadInProgress:
			downloader := NewDownloader(cacheUrl)
			response, err := downloader.Get(r)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set(`Content-Type`, response.Header.Get(`Content-Type`))
			if cacheContentDisposition := response.Header.Get(`Content-Disposition`); cacheContentDisposition != `` {
				w.Header().Set(`Content-Disposition`, cacheContentDisposition)
			}
			if contentLength := response.Header.Get(`Content-Length`); contentLength != `` {
				w.Header().Set(`Content-Length`, contentLength)
			}
			w.Header().Set(`X-Cache-State`, `download in progress`)
			DebugLog.Println(response.Header)
			if err := downloader.Download(w, nil); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		case CacheStateFound:
			if contentType := cache.ContentType; contentType != `` {
				w.Header().Set(`Content-Type`, contentType)
			}
			if cacheContentDisposition := cache.ContentDisposition; cacheContentDisposition != `` {
				w.Header().Set(`Content-Disposition`, cacheContentDisposition)
			}
			if contentLength := cache.Size(); contentLength != 0 {
				w.Header().Set(`Content-Length`, fmt.Sprintf("%d", contentLength))
			}
			w.Header().Set(`X-Cache-State`, `found`)
			io.Copy(w, cache)
		}
	}
}

func NewCore(storagePath string, auth *Auth, insecure bool) *Core {
	core := Core{
		StoragePath: storagePath,
		Insecure:    insecure,
		Auth:        auth,
		Downloaders: make(map[string]*Downloader),
	}
	return &core
}
