//! LED control for Linux using sysfs
//!
//! This module provides a high-level interface for controlling LEDs on Linux
//! systems through the sysfs interface (/sys/class/leds).

use log::info;
use std::collections::HashMap;
use std::fs::{File, OpenOptions};
use std::io::{Read, Write};
use std::path::{Path, PathBuf};
use std::sync::{Arc, Mutex};

/// Error types for LED operations
#[derive(Debug)]
pub enum LedError {
    IoError(std::io::Error),
    InvalidBrightness(u32),
    DeviceNotFound(String),
    ParseError(String),
}

impl std::fmt::Display for LedError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            LedError::IoError(e) => write!(f, "IO error: {}", e),
            LedError::InvalidBrightness(b) => write!(f, "Invalid brightness value: {}", b),
            LedError::DeviceNotFound(name) => write!(f, "LED device not found: {}", name),
            LedError::ParseError(msg) => write!(f, "Parse error: {}", msg),
        }
    }
}

impl std::error::Error for LedError {}

impl From<std::io::Error> for LedError {
    fn from(error: std::io::Error) -> Self {
        LedError::IoError(error)
    }
}

/// LED trigger types
#[derive(Debug, Clone, PartialEq)]
pub enum LedTrigger {
    None,
    Timer,
    Heartbeat,
    DiskActivity,
    NetDev,
    Custom(String),
}

impl std::fmt::Display for LedTrigger {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            LedTrigger::None => write!(f, "none"),
            LedTrigger::Timer => write!(f, "timer"),
            LedTrigger::Heartbeat => write!(f, "heartbeat"),
            LedTrigger::DiskActivity => write!(f, "disk-activity"),
            LedTrigger::NetDev => write!(f, "netdev"),
            LedTrigger::Custom(s) => write!(f, "{}", s),
        }
    }
}

/// LED information structure
#[derive(Debug, Clone)]
pub struct LedInfo {
    pub name: String,
    pub max_brightness: u32,
    pub current_brightness: u32,
    pub trigger: Option<LedTrigger>,
    pub available_triggers: Vec<LedTrigger>,
}

/// LED controller for a single LED device
pub struct Led {
    name: String,
    base_path: PathBuf,
    max_brightness: u32,
    brightness_file: Arc<Mutex<File>>,
}

impl Led {
    const BASE_PATH: &'static str = "/sys/class/leds";

    /// Create a new LED controller for the specified LED name
    ///
    /// # Arguments
    /// * `name` - The LED name (e.g., "led0", "power", "status")
    ///
    /// # Returns
    /// * `Result<Self, LedError>` - The LED controller or an error
    pub fn new(name: &str) -> Result<Self, LedError> {
        let base_path = PathBuf::from(Self::BASE_PATH).join(name);

        if !base_path.exists() {
            return Err(LedError::DeviceNotFound(name.to_string()));
        }

        let brightness_path = base_path.join("brightness");
        let max_brightness_path = base_path.join("max_brightness");

        // Read max brightness
        let max_brightness = Self::read_u32_from_file(&max_brightness_path)?;

        // Open brightness file for writing
        let brightness_file = OpenOptions::new()
            .write(true)
            .truncate(true)
            .open(&brightness_path)?;

        info!(
            "Created LED controller for '{}' with max brightness: {}",
            name, max_brightness
        );

        Ok(Self {
            name: name.to_string(),
            base_path,
            max_brightness,
            brightness_file: Arc::new(Mutex::new(brightness_file)),
        })
    }

    /// Get the LED name
    pub fn name(&self) -> &str {
        &self.name
    }

    /// Get the maximum brightness value
    pub fn max_brightness(&self) -> u32 {
        self.max_brightness
    }

    /// Get the current brightness value
    pub fn brightness(&self) -> Result<u32, LedError> {
        let brightness_path = self.base_path.join("brightness");
        Self::read_u32_from_file(&brightness_path)
    }

    /// Set the LED brightness
    ///
    /// # Arguments
    /// * `value` - Brightness value (0 to max_brightness)
    ///
    /// # Returns
    /// * `Result<(), LedError>` - Success or error
    pub fn set_brightness(&self, value: u32) -> Result<(), LedError> {
        if value > self.max_brightness {
            return Err(LedError::InvalidBrightness(value));
        }

        let mut file = self.brightness_file.lock().unwrap();
        write!(file, "{}", value)?;
        file.flush()?;

        Ok(())
    }

    /// Turn the LED on (set to maximum brightness)
    pub fn on(&self) -> Result<(), LedError> {
        self.set_brightness(self.max_brightness)
    }

    /// Turn the LED off (set brightness to 0)
    pub fn off(&self) -> Result<(), LedError> {
        self.set_brightness(0)
    }

    /// Set the LED brightness as a percentage (0.0 to 1.0)
    ///
    /// # Arguments
    /// * `percentage` - Brightness percentage (0.0 = off, 1.0 = max)
    ///
    /// # Returns
    /// * `Result<(), LedError>` - Success or error
    pub fn set_brightness_percent(&self, percentage: f32) -> Result<(), LedError> {
        if percentage < 0.0 || percentage > 1.0 {
            return Err(LedError::InvalidBrightness((percentage * 100.0) as u32));
        }

        let brightness = (percentage * self.max_brightness as f32) as u32;
        self.set_brightness(brightness)
    }

    /// Get the current LED trigger
    pub fn trigger(&self) -> Result<Option<LedTrigger>, LedError> {
        let trigger_path = self.base_path.join("trigger");

        if !trigger_path.exists() {
            return Ok(None);
        }

        let content = Self::read_string_from_file(&trigger_path)?;
        let current_trigger = Self::parse_current_trigger(&content)?;

        Ok(Some(current_trigger))
    }

    /// Set the LED trigger
    ///
    /// # Arguments
    /// * `trigger` - The trigger to set
    ///
    /// # Returns
    /// * `Result<(), LedError>` - Success or error
    pub fn set_trigger(&self, trigger: LedTrigger) -> Result<(), LedError> {
        let trigger_path = self.base_path.join("trigger");

        if !trigger_path.exists() {
            return Err(LedError::DeviceNotFound(format!(
                "trigger for {}",
                self.name
            )));
        }

        let mut file = OpenOptions::new()
            .write(true)
            .truncate(true)
            .open(&trigger_path)?;

        write!(file, "{}", trigger)?;
        file.flush()?;

        info!("Set trigger for LED '{}' to: {}", self.name, trigger);
        Ok(())
    }

    /// Get available triggers for this LED
    pub fn available_triggers(&self) -> Result<Vec<LedTrigger>, LedError> {
        let trigger_path = self.base_path.join("trigger");

        if !trigger_path.exists() {
            return Ok(vec![]);
        }

        let content = Self::read_string_from_file(&trigger_path)?;
        Self::parse_available_triggers(&content)
    }

    /// Get detailed information about this LED
    pub fn info(&self) -> Result<LedInfo, LedError> {
        Ok(LedInfo {
            name: self.name.clone(),
            max_brightness: self.max_brightness,
            current_brightness: self.brightness()?,
            trigger: self.trigger()?,
            available_triggers: self.available_triggers()?,
        })
    }

    /// Set timer trigger parameters (delay_on and delay_off in milliseconds)
    ///
    /// # Arguments
    /// * `delay_on` - Time LED is on in milliseconds
    /// * `delay_off` - Time LED is off in milliseconds
    ///
    /// # Returns
    /// * `Result<(), LedError>` - Success or error
    pub fn set_timer(&self, delay_on: u32, delay_off: u32) -> Result<(), LedError> {
        // First set the timer trigger
        self.set_trigger(LedTrigger::Timer)?;

        // Set delay_on
        let delay_on_path = self.base_path.join("delay_on");
        if delay_on_path.exists() {
            Self::write_u32_to_file(&delay_on_path, delay_on)?;
        }

        // Set delay_off
        let delay_off_path = self.base_path.join("delay_off");
        if delay_off_path.exists() {
            Self::write_u32_to_file(&delay_off_path, delay_off)?;
        }

        info!(
            "Set timer for LED '{}': on={}ms, off={}ms",
            self.name, delay_on, delay_off
        );
        Ok(())
    }

    /// List all available LEDs on the system
    pub fn list_available() -> Result<Vec<String>, LedError> {
        let base_path = Path::new(Self::BASE_PATH);

        if !base_path.exists() {
            return Err(LedError::DeviceNotFound(
                "LED subsystem not available".to_string(),
            ));
        }

        let mut leds = Vec::new();

        for entry in std::fs::read_dir(base_path)? {
            let entry = entry?;
            if entry.file_type()?.is_dir() {
                if let Some(name) = entry.file_name().to_str() {
                    leds.push(name.to_string());
                }
            }
        }

        leds.sort();
        Ok(leds)
    }

    /// Helper function to read a u32 value from a file
    fn read_u32_from_file(path: &Path) -> Result<u32, LedError> {
        let content = Self::read_string_from_file(path)?;
        content
            .trim()
            .parse::<u32>()
            .map_err(|_| LedError::ParseError(format!("Failed to parse u32 from file: {:?}", path)))
    }

    /// Helper function to read a string from a file
    fn read_string_from_file(path: &Path) -> Result<String, LedError> {
        let mut file = File::open(path)?;
        let mut content = String::new();
        file.read_to_string(&mut content)?;
        Ok(content)
    }

    /// Helper function to write a u32 value to a file
    fn write_u32_to_file(path: &Path, value: u32) -> Result<(), LedError> {
        let mut file = OpenOptions::new().write(true).truncate(true).open(path)?;
        write!(file, "{}", value)?;
        file.flush()?;
        Ok(())
    }

    /// Parse current trigger from trigger file content
    fn parse_current_trigger(content: &str) -> Result<LedTrigger, LedError> {
        // The current trigger is enclosed in square brackets
        // e.g., "none timer [heartbeat] disk-activity"
        for word in content.split_whitespace() {
            if word.starts_with('[') && word.ends_with(']') {
                let trigger_name = &word[1..word.len() - 1];
                return Ok(Self::parse_trigger_name(trigger_name));
            }
        }

        // If no current trigger found, default to none
        Ok(LedTrigger::None)
    }

    /// Parse available triggers from trigger file content
    fn parse_available_triggers(content: &str) -> Result<Vec<LedTrigger>, LedError> {
        let mut triggers = Vec::new();

        for word in content.split_whitespace() {
            let trigger_name = if word.starts_with('[') && word.ends_with(']') {
                &word[1..word.len() - 1]
            } else {
                word
            };

            triggers.push(Self::parse_trigger_name(trigger_name));
        }

        Ok(triggers)
    }

    /// Parse trigger name string into LedTrigger enum
    fn parse_trigger_name(name: &str) -> LedTrigger {
        match name {
            "none" => LedTrigger::None,
            "timer" => LedTrigger::Timer,
            "heartbeat" => LedTrigger::Heartbeat,
            "disk-activity" => LedTrigger::DiskActivity,
            "netdev" => LedTrigger::NetDev,
            _ => LedTrigger::Custom(name.to_string()),
        }
    }
}

/// LED manager for controlling multiple LEDs
pub struct LedManager {
    leds: HashMap<String, Led>,
}

impl LedManager {
    /// Create a new LED manager
    pub fn new() -> Self {
        Self {
            leds: HashMap::new(),
        }
    }

    /// Add an LED to the manager
    ///
    /// # Arguments
    /// * `name` - The LED name
    ///
    /// # Returns
    /// * `Result<(), LedError>` - Success or error
    pub fn add_led(&mut self, name: &str) -> Result<(), LedError> {
        let led = Led::new(name)?;
        self.leds.insert(name.to_string(), led);
        info!("Added LED '{}' to manager", name);
        Ok(())
    }

    /// Get a reference to an LED
    ///
    /// # Arguments
    /// * `name` - The LED name
    ///
    /// # Returns
    /// * `Option<&Led>` - Reference to the LED or None if not found
    pub fn get_led(&self, name: &str) -> Option<&Led> {
        self.leds.get(name)
    }

    /// Get a mutable reference to an LED
    ///
    /// # Arguments
    /// * `name` - The LED name
    ///
    /// # Returns
    /// * `Option<&mut Led>` - Mutable reference to the LED or None if not found
    pub fn get_led_mut(&mut self, name: &str) -> Option<&mut Led> {
        self.leds.get_mut(name)
    }

    /// Turn all LEDs on
    pub fn all_on(&self) -> Result<(), LedError> {
        for led in self.leds.values() {
            led.on()?;
        }
        Ok(())
    }

    /// Turn all LEDs off
    pub fn all_off(&self) -> Result<(), LedError> {
        for led in self.leds.values() {
            led.off()?;
        }
        Ok(())
    }

    /// Set brightness for all LEDs
    ///
    /// # Arguments
    /// * `brightness` - Brightness value (0 to max_brightness for each LED)
    ///
    /// # Returns
    /// * `Result<(), LedError>` - Success or error
    pub fn set_all_brightness(&self, brightness: u32) -> Result<(), LedError> {
        for led in self.leds.values() {
            let clamped_brightness = brightness.min(led.max_brightness());
            led.set_brightness(clamped_brightness)?;
        }
        Ok(())
    }

    /// Set brightness percentage for all LEDs
    ///
    /// # Arguments
    /// * `percentage` - Brightness percentage (0.0 to 1.0)
    ///
    /// # Returns
    /// * `Result<(), LedError>` - Success or error
    pub fn set_all_brightness_percent(&self, percentage: f32) -> Result<(), LedError> {
        for led in self.leds.values() {
            led.set_brightness_percent(percentage)?;
        }
        Ok(())
    }

    /// Get list of managed LED names
    pub fn led_names(&self) -> Vec<String> {
        self.leds.keys().cloned().collect()
    }

    /// Get information about all managed LEDs
    pub fn info_all(&self) -> Result<Vec<LedInfo>, LedError> {
        let mut infos = Vec::new();
        for led in self.leds.values() {
            infos.push(led.info()?);
        }
        Ok(infos)
    }
}

impl Default for LedManager {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_led_manager_creation() {
        let manager = LedManager::new();
        assert_eq!(manager.led_names().len(), 0);
    }

    #[test]
    fn test_list_available_leds() {
        // This test depends on the system having LEDs available
        match Led::list_available() {
            Ok(leds) => {
                // Should not panic
                println!("Available LEDs: {:?}", leds);
            }
            Err(e) => {
                // Expected if no LEDs available or no permission
                println!("LED listing error (expected): {}", e);
            }
        }
    }

    #[test]
    fn test_led_trigger_parsing() {
        let content = "none timer [heartbeat] disk-activity";
        let current = Led::parse_current_trigger(content).unwrap();
        assert_eq!(current, LedTrigger::Heartbeat);

        let available = Led::parse_available_triggers(content).unwrap();
        assert_eq!(available.len(), 4);
        assert!(available.contains(&LedTrigger::None));
        assert!(available.contains(&LedTrigger::Timer));
        assert!(available.contains(&LedTrigger::Heartbeat));
        assert!(available.contains(&LedTrigger::DiskActivity));
    }

    #[test]
    fn test_led_brightness_percentage() {
        // Test brightness percentage calculation
        // Since we can't test actual LED without hardware, we test the logic
        let max_brightness = 255;
        let percentage = 0.5;
        let expected_brightness = (percentage * max_brightness as f32) as u32;
        assert_eq!(expected_brightness, 127);
    }
}
