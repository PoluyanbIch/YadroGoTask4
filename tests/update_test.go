package api_test

import (
	"encoding/json"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type UpdateStats struct {
	WordsTotal    int `json:"words_total"`
	WordsUnique   int `json:"words_unique"`
	ComicsFetched int `json:"comics_fetched"`
	ComicsTotal   int `json:"comics_total"`
}

type UpdateStatus struct {
	Status string `json:"status"`
}

func prepare(t *testing.T) {
	req, err := http.NewRequest(http.MethodDelete, address+"/api/db", nil)
	require.NoError(t, err, "cannot make request")
	resp, err := client.Do(req)
	require.NoError(t, err, "could not send clean up command")
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	st := stats(t)
	require.Equal(t, 0, st.ComicsFetched)
	require.True(t, st.ComicsTotal > 3000, "there are more than 3000 comics in XKCD")
	require.Equal(t, 0, st.WordsTotal)
	require.Equal(t, 0, st.WordsUnique)

	require.Equal(t, "idle", status(t))
}

func TestEmptyDB(t *testing.T) {
	prepare(t)
}

func TestUpdate(t *testing.T) {
	prepare(t)
	var wg sync.WaitGroup
	wg.Add(3)
	var res1, res2 int
	var res3 string
	go func() {
		res1 = update(t)
		wg.Done()
	}()
	go func() {
		res2 = update(t)
		wg.Done()
	}()
	go func() {
		time.Sleep(1 * time.Second)
		res3 = status(t)
		wg.Done()
	}()
	wg.Wait()
	require.True(t,
		res1 == http.StatusOK && res2 == http.StatusAccepted ||
			res2 == http.StatusOK && res1 == http.StatusAccepted,
		"wrong statuses from concurrent updates, expect ok && accepted",
	)
	require.Equal(t, "running", res3, "need running status while update")
	st := stats(t)
	require.Equal(t, st.ComicsTotal, st.ComicsFetched)
	require.True(t, st.ComicsTotal > 3000, "there are more than 3000 comics in XKCD")
	require.True(t, 1000 < st.WordsTotal, "not enough total words in DB")
	require.True(t, 100 < st.WordsUnique, "not enough unique words in DB")

	prepare(t)
}

func update(t *testing.T) int {
	req, err := http.NewRequest(http.MethodPost, address+"/api/db/update", nil)
	require.NoError(t, err, "cannot make request")
	resp, err := client.Do(req)
	require.NoError(t, err, "could not send update command")
	defer resp.Body.Close()
	return resp.StatusCode
}

func status(t *testing.T) string {
	resp, err := client.Get(address + "/api/db/status")
	require.NoError(t, err, "could not get status")
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var status UpdateStatus
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&status), "cannot decode")
	return status.Status
}

func stats(t *testing.T) UpdateStats {
	resp, err := client.Get(address + "/api/db/stats")
	require.NoError(t, err, "could not get stats")
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var stats UpdateStats
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&stats), "cannot decode")
	return stats
}
