package main

import (
  "fmt"
  "io"
  "net/http"
  "strconv"
  "strings"

  "github.com/Ferlab-Ste-Justine/etcd-sdk/client"
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

	keyInfo, keyErr := cli.GetKey(statePath, client.GetKeyOptions{})
	if keyErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"error": keyErr.Error(),
		})
		return	
	}
	if !keyInfo.Found() {
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
	keyInfo, keyErr := cli.GetKey(statePath, client.GetKeyOptions{})
	if keyErr != nil {
		fmt.Printf("Could not check for legacy state: %s\n", keyErr.Error())
		return
	}
	if !keyInfo.Found() {
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
	GetHealth   gin.HandlerFunc
	Terminate   gin.HandlerFunc
}

func GetHandlers(config Config, cli *client.EtcdClient) (Handlers, <-chan struct{}) {
	terminateCh := make(chan struct{})
	
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
		putErr := cli.PutChunkedKey(&client.ChunkedKeyPayload{
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

	getHealth := func(c *gin.Context) {
		_, err := cli.GetMembers(false)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status": "error",
				"error": err.Error(),
			})
			return	
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	}

	terminate := func(c *gin.Context) {
		if c != nil {
			fmt.Println("Termination triggered via api")

			c.JSON(http.StatusOK, gin.H{
				"status": "ok",
			})
		}

		close(terminateCh)
	}

	return Handlers{
		AcquireLock: acquireLock,
		ReleaseLock: releaseLock,
		UpsertState: upsertState,
		GetState:    getState,
		DeleteState: deleteState,
		GetHealth:   getHealth,
		Terminate:   terminate,
	}, terminateCh
}