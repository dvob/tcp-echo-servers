mod pool;

use std::error::Error;
use std::io::{
    Read,
    Write
};
use std::net::{SocketAddr, TcpStream};
use std::process::exit;

enum Mode {
    ThreadPerRequest,
    ThreadPool,
    Async,
}

fn print_usage() {
    println!("Usage: echo-rs [options]");
    println!();
    println!("Flags:");
    println!(" -h       Show this help message");
    println!(" -t       Run server in thread per connection mode (default is async)");
    println!(" -p SIZE  Run server in thread pool mode (default is async)");
}

fn main() -> Result<(), Box<dyn Error>> {
    let mut mode = Mode::Async;
    let mut pool  = 5;
    let mut args = std::env::args().skip(1);
    while let Some(arg) = args.next() {
        match arg.as_str() {
            "-h" => {
                print_usage();
                return Ok(());
            }
            "-t" => {
                mode = Mode::ThreadPerRequest
            },
            "-p" => {
                mode = Mode::ThreadPool;
                let size = args.next().unwrap_or_else(||{
                    println!("missing pool size for option -p");
                    exit(1);
                });
                pool = match size.parse() {
                    Ok(pool_size) => pool_size,
                    Err(err) => {
                        println!("failed to parse pool size'{}': {}", size, err);
                        exit(1)
                    },
                };
            },
            a => {
                println!("unknown argument '{}'", a);
                exit(1);
            },
        }
    }

    match mode {
        Mode::ThreadPerRequest => {
            thread_per_request_server()?;
        },
        Mode::ThreadPool => {
            thread_pool_server(pool)?;
        },
        Mode::Async => {
            let rt = tokio::runtime::Runtime::new().unwrap();
            rt.block_on(async_server())?;
        }
    };
    Ok(())
}

fn thread_pool_server(size: u32) -> Result<(), Box<dyn Error>> {
    let pool = pool::ThreadPool::new(size);
    let listener = std::net::TcpListener::bind("127.0.0.1:1234")?;
    loop {
        let (stream, addr) = listener.accept()?;
        pool.execute(move ||{
            if let Err(err) = handle_connection(stream, addr) {
                println!("error with '{}': {}", addr, err);
            }
        });
    }
}

fn thread_per_request_server() -> Result<(), Box<dyn Error>> {
    let listener = std::net::TcpListener::bind("127.0.0.1:1234")?;
    loop {
        let (stream, addr) = listener.accept()?;
        std::thread::spawn(move || {
            if let Err(err) = handle_connection(stream, addr) {
                println!("error with '{}': {}", addr, err);
            }
        });
    }
}

fn handle_connection(mut stream: TcpStream, addr: SocketAddr) -> Result<(), Box<dyn Error>> {
            //println!("new connection from {}", &addr);
            let mut buf: [u8; 1024] = [0; 1024];
            let mut total: u32 = 0;
            while let Ok(n) = stream.read(&mut buf) {
                total += n as u32;
                if n == 0 {
                    break;
                }
                if let Err(err) = stream.write(&buf[0..n]) {
                    println!("failed to write to {}: {}", &addr, err);
                };
            }
            //println!("read {} bytes. close connection for {}", total, &addr);
            Ok(())
}

async fn async_server() -> Result<(), Box<dyn Error>> {
    let listener = tokio::net::TcpListener::bind("127.0.0.1:1234").await?;

    loop {
        let (mut stream, addr) = listener.accept().await?;
        tokio::spawn(async move {
            //println!("new connection from {}", &addr);
            let (mut r, mut w) = stream.split();
            match tokio::io::copy(&mut r, &mut w).await {
                Ok(n) => {
                    //println!("transferred {} bytes to {}", n, &addr);
                }
                Err(err) => {
                    println!("failed to transfer bytes to {}: {}", &addr, err)
                }
            }
        });
    }
}