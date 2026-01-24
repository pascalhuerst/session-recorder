//! GUI module for displaying recorder status information on a 7" touch LCD (800x480)
//!
//! This module provides an egui-based interface that displays real-time status
//! updates from multiple audio recorders, including signal status, RMS levels,
//! clipping indicators, recording time, and waveform visualization.

use eframe::egui;
use std::collections::{HashMap, VecDeque};
use std::sync::{Arc, Mutex};
use std::time::{Duration, Instant};

use crate::status_reader::common::RecorderStatus;

/// Audio configuration constants
const CHANNELS: u32 = 2;
const SAMPLE_RATE: u32 = 48000;
const WAVEFORM_DECAY_DURATION: Duration = Duration::from_secs(10);
const MAX_WAVEFORM_SAMPLES: usize = 400; // Width of waveform display
const RMS_DECAY_RATE: f32 = 0.95; // RMS level decay per frame
const CLIPPING_DISPLAY_DURATION: Duration = Duration::from_secs(5);

/// Waveform sample with timestamp for decay effect
#[derive(Clone, Debug)]
pub struct WaveformSample {
    value: f32, // Normalized sample value (-1.0 to 1.0)
    timestamp: Instant,
}

/// Recording session data for a specific recorder
#[derive(Clone, Debug)]
pub struct RecordingSession {
    pub session_id: String,
    pub start_time: Instant,
    pub total_chunks: u64,
    pub total_samples: u64,
    pub waveform_data: VecDeque<WaveformSample>,
    pub is_recording: bool,
}

/// RMS level with decay and clipping tracking
#[derive(Clone, Debug)]
pub struct RmsLevel {
    pub current_level: f32,
    pub peak_level: f32,
    pub is_clipping: bool,
    pub clipping_time: Option<Instant>,
}

impl RmsLevel {
    pub fn new() -> Self {
        Self {
            current_level: 0.0,
            peak_level: 0.0,
            is_clipping: false,
            clipping_time: None,
        }
    }

    pub fn update(&mut self, new_level: f32, clipping: bool) {
        self.current_level = new_level.max(self.current_level * RMS_DECAY_RATE);
        self.peak_level = self.peak_level.max(new_level) * RMS_DECAY_RATE;

        if clipping && !self.is_clipping {
            self.is_clipping = true;
            self.clipping_time = Some(Instant::now());
        }

        // Reset clipping after 5 seconds
        if let Some(clip_time) = self.clipping_time {
            if clip_time.elapsed() > CLIPPING_DISPLAY_DURATION {
                self.is_clipping = false;
                self.clipping_time = None;
            }
        }
    }
}

impl RecordingSession {
    pub fn new(session_id: String) -> Self {
        Self {
            session_id,
            start_time: Instant::now(),
            total_chunks: 0,
            total_samples: 0,
            waveform_data: VecDeque::with_capacity(MAX_WAVEFORM_SAMPLES),
            is_recording: true,
        }
    }

    pub fn add_chunk_data(&mut self, data: &[u32]) {
        self.total_chunks += 1;
        self.total_samples += data.len() as u64;

        // Convert samples to waveform data (take every Nth sample for visualization)
        let step = (data.len() / 20).max(1); // Sample ~20 points per chunk
        let now = Instant::now();

        for (_i, &sample) in data.iter().enumerate().step_by(step) {
            // Convert from u32 (LE 16-bit packed) to normalized f32
            let left_sample = (sample & 0xFFFF) as i16;
            let right_sample = ((sample >> 16) & 0xFFFF) as i16;

            // Use RMS of both channels for waveform
            let rms = ((left_sample as f32).powi(2) + (right_sample as f32).powi(2)).sqrt() / 2.0;
            let normalized = (rms / i16::MAX as f32).clamp(0.0, 1.0);

            self.waveform_data.push_back(WaveformSample {
                value: normalized,
                timestamp: now,
            });

            // Keep only the most recent samples
            if self.waveform_data.len() > MAX_WAVEFORM_SAMPLES {
                self.waveform_data.pop_front();
            }
        }
    }

    fn get_recording_duration(&self) -> Duration {
        if self.total_samples == 0 {
            Duration::ZERO
        } else {
            Duration::from_secs_f64(self.total_samples as f64 / (CHANNELS * SAMPLE_RATE) as f64)
        }
    }

    pub fn cleanup_old_waveform_data(&mut self) {
        let cutoff_time = Instant::now() - WAVEFORM_DECAY_DURATION;
        while let Some(sample) = self.waveform_data.front() {
            if sample.timestamp < cutoff_time {
                self.waveform_data.pop_front();
            } else {
                break;
            }
        }
    }

    pub fn stop_recording(&mut self) {
        self.is_recording = false;
    }
}

/// Extended recorder data including status, recording session, and RMS level
pub type RecorderData = (RecorderStatus, Instant, Option<RecordingSession>, RmsLevel);

/// Shared state for recorder statuses and recording data, thread-safe for concurrent access
pub type RecorderStatusMap = Arc<Mutex<HashMap<String, RecorderData>>>;

/// Main GUI application for displaying recorder statuses
/// Optimized for 800x480 touch screen display
pub struct RecorderDisplayApp {
    pub recorder_statuses: RecorderStatusMap,
    last_update_check: Instant,
}

impl Default for RecorderDisplayApp {
    fn default() -> Self {
        Self {
            recorder_statuses: Arc::new(Mutex::new(HashMap::new())),
            last_update_check: Instant::now(),
        }
    }
}

impl RecorderDisplayApp {
    pub fn new() -> Self {
        Self::default()
    }

    /// Draw waveform visualization with decay effect
    fn draw_waveform(&self, ui: &mut egui::Ui, rect: egui::Rect, session: &RecordingSession) {
        let painter = ui.painter_at(rect);

        // Draw background
        painter.rect_filled(
            rect,
            egui::Rounding::same(4.0),
            egui::Color32::from_gray(20),
        );
        painter.rect_stroke(
            rect,
            egui::Rounding::same(4.0),
            egui::Stroke::new(1.0, egui::Color32::from_gray(60)),
        );

        if session.waveform_data.is_empty() {
            // Draw center line when no data
            let center_y = rect.center().y;
            painter.line_segment(
                [
                    egui::pos2(rect.left(), center_y),
                    egui::pos2(rect.right(), center_y),
                ],
                egui::Stroke::new(1.0, egui::Color32::from_gray(40)),
            );
            return;
        }

        let now = Instant::now();
        let width = rect.width();
        let height = rect.height();
        let center_y = rect.center().y;

        // Calculate positions and draw waveform
        let samples: Vec<_> = session.waveform_data.iter().collect();
        let sample_count = samples.len();

        if sample_count > 1 {
            for i in 0..(sample_count - 1) {
                let x1 = rect.left() + (i as f32 * width) / (sample_count - 1) as f32;
                let x2 = rect.left() + ((i + 1) as f32 * width) / (sample_count - 1) as f32;

                let y1 = center_y - (samples[i].value * height * 0.4); // Scale to 40% of height
                let y2 = center_y - (samples[i + 1].value * height * 0.4);

                // Calculate decay alpha based on age
                let age = now.duration_since(samples[i].timestamp);
                let decay_factor = if age > WAVEFORM_DECAY_DURATION {
                    0.0
                } else {
                    1.0 - (age.as_secs_f32() / WAVEFORM_DECAY_DURATION.as_secs_f32())
                };

                let alpha = (decay_factor * 255.0) as u8;
                let color = if session.is_recording {
                    egui::Color32::from_rgba_unmultiplied(0, 255, 100, alpha) // Green when recording
                } else {
                    egui::Color32::from_rgba_unmultiplied(100, 100, 255, alpha) // Blue when stopped
                };

                if alpha > 10 {
                    // Only draw if visible enough
                    painter.line_segment(
                        [egui::pos2(x1, y1), egui::pos2(x2, y2)],
                        egui::Stroke::new(2.0, color),
                    );
                }
            }
        }

        // Draw center line
        painter.line_segment(
            [
                egui::pos2(rect.left(), center_y),
                egui::pos2(rect.right(), center_y),
            ],
            egui::Stroke::new(1.0, egui::Color32::from_gray(40)),
        );
    }
}

impl eframe::App for RecorderDisplayApp {
    fn update(&mut self, ctx: &egui::Context, _frame: &mut eframe::Frame) {
        // Set up continuous repainting to keep UI updated
        ctx.request_repaint_after(Duration::from_millis(100));

        // Clean up stale entries and old waveform data every 5 seconds
        if self.last_update_check.elapsed() > Duration::from_secs(5) {
            let mut statuses = self.recorder_statuses.lock().unwrap();
            let now = Instant::now();

            // Clean up old waveform data and remove stale entries
            statuses.retain(|_, (_, last_seen, recording_session, rms_level)| {
                if let Some(session) = recording_session {
                    session.cleanup_old_waveform_data();
                }
                // Update RMS decay
                rms_level.current_level *= RMS_DECAY_RATE;
                rms_level.peak_level *= RMS_DECAY_RATE;

                now.duration_since(*last_seen) < Duration::from_secs(30) // Remove after 30 seconds of no updates
            });
            self.last_update_check = now;
        }

        // Ensure consistent scaling for touch screen usability
        ctx.set_pixels_per_point(1.25);

        // Configure for embedded/kiosk mode
        //ctx.send_viewport_cmd(egui::ViewportCommand::Fullscreen(true));

        egui::CentralPanel::default().show(ctx, |ui| {
            // Compact header
            ui.horizontal(|ui| {
                ui.label(egui::RichText::new("Session Recorder Status Display").size(18.0));
            });
            ui.add_space(8.0);

            let statuses = self.recorder_statuses.lock().unwrap();

            if statuses.is_empty() {
                ui.centered_and_justified(|ui| {
                    ui.label(egui::RichText::new("No recorders connected").size(18.0));
                });
                return;
            }

            // Create a responsive layout without scrollbar
            let available_width = ui.available_width();
            let available_height = ui.available_height();
            let card_width = 380.0;
            let card_height = 180.0; // Reduced height
            let columns = ((available_width / card_width) as usize).max(1);
            let _rows = (((available_height - 40.0) / (card_height + 10.0)) as usize).max(1);

            let mut current_column = 0;

            ui.horizontal_wrapped(|ui| {
                for (_recorder_id, (status, last_seen, recording_session, rms_level)) in
                    statuses.iter()
                {
                    if current_column >= columns {
                        current_column = 0;
                    }

                    // Status card with touch-friendly sizing
                    egui::Frame::none()
                        .fill(egui::Color32::from_gray(40))
                        .stroke(egui::Stroke::new(2.0, egui::Color32::from_gray(60)))
                        .rounding(egui::Rounding::same(12.0))
                        .inner_margin(egui::Margin::same(12.0))
                        .show(ui, |ui| {
                            ui.set_min_width(card_width - 24.0);
                            ui.set_min_height(card_height); // Reduced height

                            // Top row: Recorder name (large, left) and timestamp (small, right)
                            ui.horizontal(|ui| {
                                ui.label(
                                    egui::RichText::new(&status.recorder_name)
                                        .size(20.0)
                                        .strong()
                                        .color(egui::Color32::WHITE),
                                );
                                ui.with_layout(
                                    egui::Layout::right_to_left(egui::Align::Center),
                                    |ui| {
                                        let age = Instant::now().duration_since(*last_seen);
                                        let age_color = if age.as_secs() > 10 {
                                            egui::Color32::RED
                                        } else if age.as_secs() > 5 {
                                            egui::Color32::YELLOW
                                        } else {
                                            egui::Color32::GREEN
                                        };
                                        ui.colored_label(
                                            age_color,
                                            egui::RichText::new(format!("{}s ago", age.as_secs()))
                                                .size(10.0),
                                        );
                                    },
                                );
                            });

                            // UUID smaller and less bright
                            ui.label(
                                egui::RichText::new(&status.recorder_id)
                                    .size(9.0)
                                    .color(egui::Color32::GRAY),
                            );

                            ui.add_space(2.0);

                            // Main content in horizontal layout: status + level meter on left, waveform on right
                            ui.horizontal(|ui| {
                                // Left side: Status and level meter
                                ui.vertical(|ui| {
                                    ui.set_width(100.0);

                                    // Recording status with dot
                                    let is_recording = recording_session
                                        .as_ref()
                                        .map_or(false, |s| s.is_recording);
                                    let (status_text, status_color, dot_color) = if is_recording {
                                        (
                                            "Recording",
                                            egui::Color32::from_rgb(255, 80, 80),
                                            egui::Color32::RED,
                                        )
                                    } else {
                                        ("Idle", egui::Color32::GRAY, egui::Color32::DARK_GRAY)
                                    };

                                    ui.horizontal(|ui| {
                                        let dot_rect = ui.allocate_space(egui::vec2(8.0, 8.0)).1;
                                        ui.painter().circle_filled(
                                            dot_rect.center(),
                                            4.0,
                                            dot_color,
                                        );
                                        ui.add_space(2.0);
                                        ui.colored_label(
                                            status_color,
                                            egui::RichText::new(status_text).size(12.0),
                                        );
                                    });

                                    ui.add_space(2.0);

                                    // Vertical level meter
                                    ui.horizontal(|ui| {
                                        ui.label(egui::RichText::new("Level").size(9.0));

                                        let meter_width = 10.0;
                                        let meter_height = 50.0;
                                        let meter_rect = ui
                                            .allocate_space(egui::vec2(meter_width, meter_height))
                                            .1;

                                        // Draw meter background
                                        ui.painter().rect_filled(
                                            meter_rect,
                                            egui::Rounding::same(2.0),
                                            egui::Color32::from_gray(20),
                                        );

                                        // Draw level fill (from bottom up)
                                        let level_ratio =
                                            (rms_level.current_level / 100.0).clamp(0.0, 1.0);
                                        if level_ratio > 0.0 {
                                            let fill_height = level_ratio * meter_height;
                                            let fill_rect = egui::Rect::from_min_size(
                                                egui::pos2(
                                                    meter_rect.min.x,
                                                    meter_rect.max.y - fill_height,
                                                ),
                                                egui::vec2(meter_width, fill_height),
                                            );

                                            let level_color = if rms_level.is_clipping {
                                                egui::Color32::RED
                                            } else if rms_level.current_level > 80.0 {
                                                egui::Color32::YELLOW
                                            } else {
                                                egui::Color32::GREEN
                                            };

                                            ui.painter().rect_filled(
                                                fill_rect,
                                                egui::Rounding::same(2.0),
                                                level_color,
                                            );
                                        }

                                        // Draw border
                                        ui.painter().rect_stroke(
                                            meter_rect,
                                            egui::Rounding::same(2.0),
                                            egui::Stroke::new(1.0, egui::Color32::from_gray(60)),
                                        );
                                    });
                                });

                                // Right side: Session info and waveform
                                ui.vertical(|ui| {
                                    let remaining_width = ui.available_width();

                                    // Session info (compact)
                                    if let Some(session) = recording_session {
                                        let duration = session.get_recording_duration();
                                        let minutes = duration.as_secs() / 60;
                                        let seconds = duration.as_secs() % 60;
                                        let millis = (duration.subsec_millis() / 100) % 10;

                                        ui.horizontal(|ui| {
                                            ui.label(egui::RichText::new("Session:").size(9.0));
                                            ui.label(
                                                egui::RichText::new(
                                                    &session.session_id
                                                        [..8.min(session.session_id.len())],
                                                )
                                                .size(9.0)
                                                .color(egui::Color32::LIGHT_GRAY),
                                            );
                                        });

                                        ui.horizontal(|ui| {
                                            ui.label(egui::RichText::new("Time:").size(9.0));
                                            ui.colored_label(
                                                egui::Color32::LIGHT_BLUE,
                                                egui::RichText::new(format!(
                                                    "{}:{:02}.{}",
                                                    minutes, seconds, millis
                                                ))
                                                .size(9.0),
                                            );
                                        });

                                        ui.add_space(2.0);

                                        // Waveform spanning full remaining width
                                        let waveform_height = 35.0;
                                        let waveform_rect = ui
                                            .allocate_space(egui::vec2(
                                                remaining_width,
                                                waveform_height,
                                            ))
                                            .1;

                                        if ui.is_rect_visible(waveform_rect) {
                                            self.draw_waveform(ui, waveform_rect, session);
                                        }
                                    }
                                });
                            });
                        });

                    ui.add_space(8.0);
                    current_column += 1;
                }
            });
        });
    }
}

/// Runs the GUI application optimized for 7" touch LCD (800x480)
///
/// # Arguments
/// * `recorder_statuses` - Shared state containing recorder status data
///
/// # Returns
/// * `eframe::Result<()>` - Result of running the GUI application
pub fn run_gui(recorder_statuses: RecorderStatusMap) -> eframe::Result<()> {
    let mut app = RecorderDisplayApp::new();
    app.recorder_statuses = recorder_statuses;

    let options = eframe::NativeOptions {
        viewport: egui::ViewportBuilder::default()
            .with_inner_size([800.0, 480.0]) // Perfect for 7" touch LCD
            .with_min_inner_size([800.0, 480.0])
            .with_max_inner_size([800.0, 480.0])
            .with_resizable(true)
            .with_decorations(true) // No window decorations for embedded display
            .with_fullscreen(false) // Fullscreen for embedded use
            .with_always_on_top()
            .with_maximized(false),
        hardware_acceleration: eframe::HardwareAcceleration::Required,
        ..Default::default()
    };

    eframe::run_native(
        "Session Recorder Display",
        options,
        Box::new(|cc| {
            // Configure for Wayland/embedded use
            cc.egui_ctx.set_pixels_per_point(1.2);
            cc.egui_ctx.set_visuals(egui::Visuals::dark());

            // Optimize for touch input
            cc.egui_ctx.style_mut(|style| {
                style.interaction.resize_grab_radius_side = 8.0;
                style.interaction.resize_grab_radius_corner = 12.0;
                style.spacing.button_padding = egui::vec2(12.0, 8.0);
                style.spacing.item_spacing = egui::vec2(8.0, 6.0);
                style.spacing.menu_margin = egui::Margin::same(8.0);
            });

            Box::new(app)
        }),
    )
}
