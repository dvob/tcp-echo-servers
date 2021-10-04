use std::{
    thread,
    sync::{
        Arc,
        Mutex,
        mpsc::{
            Receiver,
            SyncSender
        }
    }
};

pub struct ThreadPool {
    workers: Vec<Worker>,
    sender: SyncSender<Message>,
}

impl ThreadPool {
    pub fn new(size: u32) -> Self {
        let (sender, receiver) = std::sync::mpsc::sync_channel(100);
        let receiver = Arc::new(Mutex::new(receiver));
        let mut workers = vec![];
        for i in 0..size {
            let w = Worker::new(i, Arc::clone(&receiver));
            workers.push(w);
        }
        Self {
            workers,
            sender,
        }
    }

    pub fn execute<F>(&self, f: F)
    where
        F: FnOnce() + Send + 'static
    {
        self.sender.send(Message::NewJob(Box::new(f))).unwrap();
    }
}

impl Drop for ThreadPool {
    fn drop(&mut self) {
        for _ in &self.workers {
            self.sender.send(Message::Shutdown).unwrap();
        }
        for worker in &mut self.workers {
            if let Some(thread) = worker.thread.take() {
                thread.join().unwrap();
            }
        }
    }
}

struct Worker {
    thread: Option<thread::JoinHandle<()>>,
}

type Job = Box<dyn FnOnce() + Send + 'static>;
enum Message {
    NewJob(Job),
    Shutdown,
}

impl Worker {
    fn new(id: u32, receiver: Arc<Mutex<Receiver<Message>>>) -> Self {
        let thread = thread::spawn(move ||{
            loop {
                let job = receiver.lock().unwrap().recv().unwrap();
                match job {
                    Message::NewJob(job) => job(),
                    Message::Shutdown => break,
                }
            }
        });
        Self{
            thread: Some(thread),
        }
    }
}
