package main

import (
	"flag"
	"fmt"
	"github.com/go-redis/redis"
	"github.com/gosuri/uiprogress"
	"github.com/olekukonko/tablewriter"
	log "github.com/sirupsen/logrus"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type stat struct {
	ByteCount int64
	KeyCount  int64
}

func main() {

	var (
		addr         = flag.String("addr", "127.0.0.1:6379", "redis host:port")
		password     = flag.String("password", "", "redis connection password")
		dbs          = flag.String("db", "0", "specific redis db, or a comma separated db list")
		chunkSize    = flag.Int64("chunk", 10000, "chunk size of key to analyse at once")
		segmentLimit = flag.Int64("breakdown", 2, "breakdown count")
		separator    = flag.String("separator", ":", "key prefix separator")
		match        = flag.String("match", "*", "key name filter on scan")
	)

	flag.Parse()
	uiprogress.Start()

	databases := strings.Split(*dbs, ",")

	var wg sync.WaitGroup
	var tm sync.Mutex
	maxTotal := 0

	t := tablewriter.NewWriter(os.Stdout)
	t.SetAutoMergeCells(true)

	for _, dbNum := range databases {
		wg.Add(1)
		db, _ := strconv.ParseInt(dbNum, 10, 32)
		go func(db int64) {

			keyStats := map[string]*stat{}
			defer wg.Done()

			redisClient := redis.NewClient(&redis.Options{
				Addr:        *addr,
				Password:    *password,
				DB:          int(db),
				ReadTimeout: -1 * time.Second,
			},
			)

			defer redisClient.Close()

			_, err := redisClient.Ping().Result()
			if err != nil {
				log.Errorf("error connecting db %d : %s\n", db, err)
				return
			}
			dbSize, _ := redisClient.DBSize().Result()
			if dbSize == 0 {
				log.Warnf("db %d is empty\n", db)
				return
			}
			maxTotal = int(math.Max(float64(dbSize), float64(maxTotal)))
			bar := uiprogress.AddBar(int(dbSize)) // Add a new bar

			// optionally, append and prepend completion and elapsed time
			bar.AppendCompleted()
			bar.PrependElapsed()

			crs := uint64(0)
			totalScanned := 0

			bar.PrependFunc(func(b *uiprogress.Bar) string {
				return fmt.Sprintf("[db %d] \t %"+strconv.Itoa(len(strconv.Itoa(maxTotal)))+"d/%d\t", db, b.Current(), b.Total)
			})

			for {
				keys, cursor, err := redisClient.Scan(crs, *match, *chunkSize).Result()
				totalScanned += len(keys)
				_ = bar.Set(totalScanned)
				if err != nil {
					log.Fatalf("scanning failed: %v", err)
				}
				crs = cursor
				for _, key := range keys {
					size, _ := redisClient.MemoryUsage(key).Result()
					Increase(keyStats, key, size, *segmentLimit, *separator)
				}

				if crs == 0 {
					break // end of scanning
				}
			}

			tm.Lock()
			for k, v := range keyStats {
				t.Append([]string{fmt.Sprintf("%d", db), k, fmt.Sprintf("%d", v.KeyCount), ByteCountBinary(v.ByteCount)})
			}
			t.Append([]string{"", "", "", ""})
			tm.Unlock()

		}(db)
	}
	wg.Wait()
	t.SetHeader([]string{"DB", "Prefix", "Count", "Size"})
	t.SetAutoFormatHeaders(false)
	t.SetColumnAlignment([]int{tablewriter.ALIGN_LEFT, tablewriter.ALIGN_LEFT, tablewriter.ALIGN_RIGHT, tablewriter.ALIGN_RIGHT})
	t.Render()

}

func ByteCountBinary(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

func Increase(keyStats map[string]*stat, key string, increaseBy int64, keySegmentLimit int64, separator string) {
	keySegments := strings.Split(key, separator)
	segmentLimit := int(math.Min(float64(keySegmentLimit), float64(len(keySegments))))
	segment := ""
	for i := 0; i < segmentLimit; i++ {
		if segment == "" {
			segment = keySegments[i]
		} else {
			segment = segment + separator + keySegments[i]
		}
		if _, ok := keyStats[segment]; ok {
			keyStats[segment].ByteCount += increaseBy
			keyStats[segment].KeyCount += 1
		} else {
			keyStats[segment] = &stat{increaseBy, 1}
		}
	}
}
