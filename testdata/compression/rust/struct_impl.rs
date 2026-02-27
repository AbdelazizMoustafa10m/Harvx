/// A configuration for the application.
#[derive(Debug, Clone)]
pub struct Config {
    pub host: String,
    pub port: u16,
    timeout: Duration,
}

/// A simple marker struct with no fields.
pub struct Marker;

/// A newtype wrapper around String.
pub struct Name(pub String);

impl Config {
    /// Create a new Config with defaults.
    pub fn new() -> Self {
        Config {
            host: "localhost".to_string(),
            port: 8080,
            timeout: Duration::from_secs(30),
        }
    }

    /// Set the host.
    pub fn with_host(mut self, host: &str) -> Self {
        self.host = host.to_string();
        self
    }

    /// Get the address string.
    pub fn addr(&self) -> String {
        format!("{}:{}", self.host, self.port)
    }

    fn validate(&self) -> Result<(), String> {
        if self.port == 0 {
            return Err("port cannot be 0".to_string());
        }
        Ok(())
    }
}

impl Display for Config {
    fn fmt(&self, f: &mut Formatter) -> fmt::Result {
        write!(f, "{}:{}", self.host, self.port)
    }
}