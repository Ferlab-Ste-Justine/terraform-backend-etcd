package main

import (
  "fmt"
  "io"
  "net/http"
  "os"
  "strconv"
  "strings"

  "github.com/Ferlab-Ste-Justine/etcd-sdk/client"
  "github.com/Ferlab-Ste-Justine/etcd-sdk/keymodels"
  "github.com/gin-gonic/gin"
)

func getLegacyStatePath(state string, config Config) string {
	if config.LegacySupport.AddSlash {
		return fmt.Sprintf("%s/default", state)
	}

	return fmt.Sprintf("%sdefault", state)
}

func getLegacyState(c *gin.Context, cli *client.EtcdClient, config Config) {
	statePath := getLegacyStatePath(c.Query("state"), config)

	keyInfo, keyExists, keyErr := cli.GetKey(statePath)
	if keyErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"error": keyErr.Error(),
		})
		return	
	}
	if !keyExists {
		c.JSON(http.StatusNotFound, gin.H{
			"status": "not found",
		})
		return
	}

	fmt.Printf("Reading from legacy state at: %s\n", statePath)
	c.DataFromReader(
		http.StatusOK,
		int64(len(keyInfo.Value)),
		"application/json",
		io.NopCloser(strings.NewReader(keyInfo.Value)),
		map[string]string{},
	)
}

func clearLegacyState(c *gin.Context, cli *client.EtcdClient, config Config) {
	statePath := getLegacyStatePath(c.Query("state"), config)
	_, keyExists, keyErr := cli.GetKey(statePath)
	if keyErr != nil {
		fmt.Printf("Could not check for legacy state: %s\n", keyErr.Error())
		return
	}
	if !keyExists {
		return
	}

	fmt.Printf("Clearing legacy state found at: %s\n", statePath)
	deleteErr := cli.DeleteKey(statePath)
	if deleteErr != nil {
		fmt.Printf("Could not clear legacy state: %s\n", deleteErr.Error())
		return
	}
}

type Handlers struct{
	AcquireLock gin.HandlerFunc
	ReleaseLock gin.HandlerFunc
	UpsertState gin.HandlerFunc
	GetState    gin.HandlerFunc
	DeleteState gin.HandlerFunc
	Terminate   gin.HandlerFunc
}

func GetHandlers(config Config, cli *client.EtcdClient) Handlers {
	acquireLock := func(c *gin.Context) {
		state := c.Query("state")
		if state == "" {
			c.JSON(http.StatusBadRequest , gin.H{
				"error": "State query parameter is missing",
			})
			return		
		}
		
		leaseTtlStr := c.DefaultQuery("lease_ttl", "600")
		ttl, ttlErr := strconv.ParseInt(leaseTtlStr, 10, 64)
		if ttlErr != nil {
			c.JSON(http.StatusBadRequest , gin.H{
				"error": "Lease ttl needs to be in integer format",
			})
			return		
		}
		
		state = fmt.Sprintf("%s/lock", state)
		_, alreadyLocked, lockErr := cli.AcquireLock(client.AcquireLockOptions{
			Key: state,
			Ttl: ttl,
			Timeout: config.Lock.Timeout,
			RetryInterval: config.Lock.RetryInterval,
		})
		if alreadyLocked {
			c.JSON(http.StatusLocked, gin.H{
				"status": "locked",
			})
			return		
		}
		if lockErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": "error",
				"error": lockErr.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	}

	releaseLock := func(c *gin.Context) {
		state := c.Query("state")
		if state == "" {
			c.JSON(http.StatusBadRequest , gin.H{
				"error": "State query parameter is missing",
			})
			return		
		}

		state = fmt.Sprintf("%s/lock", state)
		releaseErr := cli.ReleaseLock(state)
		if releaseErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": "error",
				"error": releaseErr.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	}

	upsertState := func(c *gin.Context) {
		state := c.Query("state")
		if state == "" {
			c.JSON(http.StatusBadRequest , gin.H{
				"error": "State query parameter is missing",
			})
			return		
		}

		state = fmt.Sprintf("%s/state", state)
		putErr := cli.PutChunkedKey(&keymodels.ChunkedKeyPayload{
			Key: state,
			Value: c.Request.Body,
			Size: c.Request.ContentLength,
		})
		if putErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": "error",
				"error": putErr.Error(),
			})
			return
		}

		if config.LegacySupport.Clear {
			clearLegacyState(c, cli, config)
		}

		c.JSON(http.StatusOK, gin.H{
			"state": state,
		})
	}

	getState := func(c *gin.Context) {
		state := c.Query("state")
		if state == "" {
			c.JSON(http.StatusBadRequest , gin.H{
				"error": "State query parameter is missing",
			})
			return		
		}

		state = fmt.Sprintf("%s/state", state)
		payload, getErr := cli.GetChunkedKey(state)
		if getErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": "error",
				"error": getErr.Error(),
			})
			return
		}

		//No data
		if payload == nil {
			if config.LegacySupport.Read {
				getLegacyState(c, cli, config)
				return
			}

			c.JSON(http.StatusNotFound, gin.H{
				"status": "not found",
			})
			return
		}

		defer payload.Close()
		c.DataFromReader(http.StatusOK, payload.Size, "application/json", payload, map[string]string{}) 
	}

	deleteState := func(c *gin.Context) {
		state := c.Query("state")
		if state == "" {
			c.JSON(http.StatusBadRequest , gin.H{
				"error": "State query parameter is missing",
			})
			return		
		}

		state = fmt.Sprintf("%s/state", state)
		deleteErr := cli.DeleteChunkedKey(state)
		if deleteErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": "error",
				"error": deleteErr.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"state": state,
		})
	}

	terminate := func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})

		fmt.Println("Termination triggered via api")
		os.Exit(0)
	}

	return Handlers{
		AcquireLock: acquireLock,
		ReleaseLock: releaseLock,
		UpsertState: upsertState,
		GetState:    getState,
		DeleteState: deleteState,
		Terminate:   terminate,
	}
}