# tcp-echo-servers
To learn Rust and its async features I wrote a TCP echo server using the Tokio runtime. To compare it to other approaches I also did an implementation which uses threads and one using Go.
* Rust using async with Tokio (`echo-rs`)
* Rust using threads (`echo-rs -t`)
* Go (`echo-go`)

To compare these solutions I wrote a benchmark tool (`bench`) which over a period of time sends and reads as many bytes as posbile.

## Results
The benchmark were run for two minutes (`-d 2m`) for each server with different settings.

### 10000 connections / ∞ requests per connection
```
./bench -c 10000 -d 2m
```
| | req/s | conn/s | avg | min | max | p95 | p99 | max rss | cpu user | cpu system |
| - | - | - | - | - | - | - | - | - | - | - |
| async | 97931.12 | 83.33 | 102.042658ms | 55µs | 469.3017ms | 237.111163ms | 298.273261ms | 25504 | 24.08 | 327.17 |
| thread | 115066.91 | 83.33 | 85.78983ms | 40.542µs | 14.757013885s | 547.585526ms | 622.210612ms | 87728 | 13.41 | 300.04 |
| golang | 86984.06 | 83.33 | 114.833644ms | 41.876µs | 481.651804ms | 145.02772ms | 165.925313ms | 41824 | 34.38 | 368.45 |

### 10000 connections / 50 requests per connection
```
./bench -c 10000 -d 2m -r 50
```
| | req/s | conn/s | avg | min | max | p95 | p99 | max rss | cpu user | cpu system |
| - | - | - | - | - | - | - | - | - | - | - |
| async | 88647.34 | 1817.14 | 111.213274ms | 55.042µs | 427.97756ms | 217.252496ms | 260.076564ms | 35032 | 24.62 | 319.05 |
| thread | 84662.59 | 1734.56 | 114.230653ms | 41.083µs | 3.632305117s | 169.661136ms | 185.01735ms | 104456 | 18.49 | 326.04 |
| golang | 77837.44 | 1584.01 | 124.764215ms | 50.376µs | 873.672437ms | 165.413559ms | 353.623351ms | 51932 | 32.53 | 354.30 |

### 10000 connections / 5 requests per connection
```
./bench -c 10000 -d 2m -r 5
```
| | req/s | conn/s | avg | min | max | p95 | p99 | max rss | cpu user | cpu system |
| - | - | - | - | - | - | - | - | - | - | - |
| async | 34452.20 | 6930.93 | 244.767562ms | 55.334µs | 879.986699ms | 618.513457ms | 684.257926ms | 41236 | 14.11 | 172.53 | 
| thread\* | 28834.06 | 5836.77 | 48.486708ms | 45.792µs | 1m56.906909275s | 97.655322ms | 486.749348ms | 34608 | 19.05 | 211.98 |
| golang | 33412.39 | 6713.61 | 250.116182ms | 50.417µs | 863.148724ms | 588.626955ms | 664.505552ms | 54744 | 26.87 | 222.66 |

\* did not run properly: many `connection timed out` and `connection reset by peer` errors. finished only after 3min. probably because it hang in the timeout.

### 5000 connections / 1 requests per connection
```
./bench -c 5000 -d 2m -r 5
```
| | req/s | conn/s | avg | min | max | p95 | p99 | max rss | cpu user | cpu system |
| - | - | - | - | - | - | - | - | - | - | - |
| async | 11671.52 | 11671.52 | 213.05352ms | 52.667µs | 858.359851ms | 690.297686ms | 721.771742ms | 28536 | 12.62 | 109.22 |
| thread\* | 10302.60 | 10306.94 | 61.041962ms | 59.917µs | 1m38.978638946s | 147.215365ms | 950.083664ms | 14052 | 20.94 | 201.78 |
| golang | 11436.03 | 11436.03 | 231.039444ms | 50.959µs | 816.143537ms | 665.963564ms | 691.388862ms | 31764 | 27.70 | 156.19 |

\* did not run properly: many `connection timed out` and `connection reset by peer` errors. finished only after 3min. probably because it hang in the timeout.
