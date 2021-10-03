# tcp-echo-servers
To learn Rust and its async features I wrote a TCP echo server using the Tokio runtime. To compare it to other approaches I also did an implementation which uses threads and one using Go.
* Rust using async with Tokio (`echo-rs`)
* Rust using threads (`echo-rs -t`)
* Go (`echo-go`)

To compare these solutions I wrote a benchmark tool (`bench`) which over a period of time sends and reads as many bytes as posbile.
