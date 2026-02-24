use std::collections::HashMap;
use std::io::{self, Read, Write};
use std::sync::{Arc, Mutex, RwLock};
use std::path::PathBuf;

// Wildcard import
use std::prelude::v1::*;

// Aliased import
use std::collections::HashMap as Map;

// Re-exports
pub use crate::config::Config;
pub use crate::error::{Error, Result};

// Crate-level re-export
pub(crate) use crate::internal::Helper;

// Super-level re-export
pub(super) use super::parent::Module;

// Deep nested use
use tokio::io::AsyncReadExt;

// Module declarations
pub mod config;
mod internal;

fn main() {
    println!("hello");
}