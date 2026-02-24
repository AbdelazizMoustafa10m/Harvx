/// Represents the status of a task.
#[derive(Debug, Clone, PartialEq)]
pub enum Status {
    /// The task is pending.
    Pending,
    /// The task is running with a progress percentage.
    Running(f64),
    /// The task completed successfully.
    Completed { output: String, duration: Duration },
    /// The task failed.
    Failed(Box<dyn std::error::Error>),
}

/// Application error type.
#[derive(Debug, thiserror::Error)]
pub enum AppError {
    #[error("IO error: {0}")]
    Io(#[from] std::io::Error),
    #[error("Parse error: {msg}")]
    Parse { msg: String, line: usize },
    #[error("Not found")]
    NotFound,
}

/// A simple color enum.
pub enum Color {
    Red,
    Green,
    Blue,
    Custom(u8, u8, u8),
}