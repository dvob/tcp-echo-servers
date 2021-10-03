# tcp-echo-servers
To learn Rust and its async features I wrote a TCP echo server using the Tokio runtime. To compare it to other approaches I also did an implementation which uses threads and one using Go.
* Rust using async with Tokio (`echo-rs`)
* Rust using threads (`echo-rs -t`)
* Go (`echo-go`)

To compare these solutions I wrote a benchmark tool (`bench`) which over a period of time sends and reads as many bytes as posbile.

# Results
## echo-rs (async)
```
$ /usr/bin/time -f "user=%U system=%S max_rss=%M" ./echo-rs/target/release/echo-rs
user=13.75 system=174.12 max_rss=25208
```
```
$ ./bench -c 10000 -d 60s
119162.03 req/s
reqest per connection: min=587, max=894, avg=714
```

## echo-rs (threading)
```
$ /usr/bin/time -f "user=%U system=%S max_rss=%M" ./echo-rs/target/release/echo-rs -t
user=8.02 system=180.37 max_rss=89432
```
```
$ ./bench -c 10000 -d 60s
126684.93 req/s
reqest per connection: min=246, max=3955, avg=760
```

## echo-go
```
$ /usr/bin/time -f "user=%U system=%S max_rss=%M" ./echo-go/echo-go
user=17.95 system=203.99 max_rss=38916
```
```
$ ./bench -c 10000 -d 60s
94023.70 req/s
reqest per connection: min=553, max=588, avg=564
```
