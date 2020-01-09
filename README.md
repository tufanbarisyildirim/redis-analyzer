# Redis Analyzer

Redis Analyzer is a simple tool to group keys by prefixes and sum their size in memory

`requires go 1.13`

### Usage
```shell script
Usage of ./bin/redis-analyzer:
  -addr string
        redis host:port (default "127.0.0.1:6379")
  -breakdown int
        breakdown count (default 2)
  -chunk int
        chunk size of key to analyse at once (default 10000)
  -db string
        specific redis db, or a comma separated db list (default "0")
  -match string
        key name filter on scan (default "*")
  -password string
        redis connection password
  -separator string
        key prefix separator (default ":")
```

### Example
``` shell script
➜  redis-analyzer ./redis-analyzer --addr=127.0.0.1:6379 --breakdown=2 --db=1,2,3,4 
db 3 is empty
db 4 is empty
[db 1]     224/224         --- [====================================================================] 100%
[db 2]   19833/19833       --- [====================================================================] 100%

+----+-----------------------+-------+-----------+
| DB |        Prefix         | Count |   Size    |
+----+-----------------------+-------+-----------+
| 1  | mytestproj            |   224 | 393.1 KiB |
|    | mytestproj:job-status |   107 |  14.5 KiB |
|    | mytestproj:job        |       |  39.1 KiB |
|    | mytestproj:all-jobs   |     1 | 336.9 KiB |
|    | mytestproj:stats      |     8 |   2.0 KiB |
|    | mytestproj:job-queue  |     1 |     552 B |
|    |                       |       |           |
| 2  | testa                 | 19833 |   7.0 MiB |
|    | testa:autlist         |       |           |
|    |                       |       |           |
+----+-----------------------+-------+-----------+

```