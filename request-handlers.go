package main

import (
  "fmt"
  "net/http"
  "strconv"

  "github.com/Ferlab-Ste-Justine/etcd-sdk/client"
  "github.com/Ferlab-Ste-Justine/etcd-sdk/keymodels"
  "github.com/gin-gonic/gin"
)

type Handlers struct{
	AcquireLock gin.HandlerFunc
	ReleaseLock gin.HandlerFunc
	UpsertState gin.HandlerFunc
	GetState    gin.HandlerFunc
	DeleteState gin.HandlerFunc
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

	return Handlers{
		AcquireLock: acquireLock,
		ReleaseLock: releaseLock,
		UpsertState: upsertState,
		GetState:    getState,
		DeleteState: deleteState,
	}
}