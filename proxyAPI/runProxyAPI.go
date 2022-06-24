package proxyAPI

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"proxy_project/cache"
	"proxy_project/handler"
	"proxy_project/proxyAPI/requestAPI"
	"strings"
)

func ProxyAPI(cacheClient cache.CacheClient, request requestAPI.RequestClient) {

	var mutex handler.QueueMutex

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		body, err := io.ReadAll(r.Body)
		if err != nil {
			if err.Error() != "EOF" {
				log.Fatal(err)
			}
		}

		queryParms := transformMapInQueryParams(r.URL.Query())
		log.Println(r.Method)

		unlock := mutex.Lock(r, body, queryParms)
		defer unlock()

		if r.Method == "GET" {
			responseCache, err := handler.FindInCache(cacheClient, r, queryParms)

			switch e := err.(type) {
			case *cache.CacheNotFoundError:
				log.Println(e)

			case *json.UnmarshalTypeError:
				responseCache, err := handler.FindInCacheArray(cacheClient, r, queryParms)
				log.Println("Array cache")

				switch e := err.(type) {
				case *cache.CacheNotFoundError:
					log.Println(e)

				default:
					if err := responseCache.ResponseWriter(w); err != nil {
						log.Fatal(err)
					}

					return
				}
			default:
				if err := responseCache.ResponseWriter(w); err != nil {
					log.Fatal(err)
				}
				log.Println("Cache")

				return

			}
		}

		log.Println("Requisition")

		response, responseArray := request.RequestToAPI(cacheClient, r, body, queryParms)

		if response != nil {
			if err := (*response).ResponseWriter(w); err != nil {
				log.Fatal(err)
			}
		} else {
			if err := (*responseArray).ResponseWriter(w); err != nil {
				log.Fatal(err)
			}
		}

		return
	})

	log.Fatal(http.ListenAndServe(":8080", nil))

}

func transformMapInQueryParams(query map[string][]string) string {
	if len(query) == 0 {
		return ""
	}

	queryContent := new(bytes.Buffer)
	fmt.Fprintf(queryContent, "?")
	for key, value := range query {
		fmt.Fprintf(queryContent, "%s=%s&", key, strings.Join(value, ""))
	}

	return queryContent.String()[:len(queryContent.String())-1]
}
