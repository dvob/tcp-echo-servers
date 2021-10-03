use std::error::Error;
use std::io::{Read, Write};

enum Mode {
    Threading,
    Async,
}

fn print_usage() {
    println!("Usage: echo-rs [options]");
    println!();
    println!("Flags:");
    println!(" -h    Show this help message");
    println!(" -t    Run server in threading mode (default is async)");
}

fn main() -> Result<(), Box<dyn Error>> {
    let mut mode = Mode::Async;
    let args = std::env::args().skip(1);
    for arg in args {
        match arg.as_str() {
            "-h" => {
                print_usage();
                return Ok(());
            }
            "-t" => {
                mode = Mode::Threading
            }
            a => {
                println!("unknown argument '{}'", a);
                std::process::exit(1);
            },
        }
    }

    match mode {
        Mode::Threading => {
            threading_server()?;
        }
        Mode::Async => {
            let rt = tokio::runtime::Runtime::new().unwrap();
            rt.block_on(async_server())?;
        }
    };
    Ok(())
}

fn threading_server() -> Result<(), Box<dyn Error>> {
    let listener = std::net::TcpListener::bind("127.0.0.1:1234")?;
    loop {
        let (mut stream, addr) = listener.accept()?;
        std::thread::spawn(move || {
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
        });
    }
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