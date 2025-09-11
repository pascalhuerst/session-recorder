//! GUI module for displaying recorder status information on a 7" touch LCD (800x480)
//!
//! This module provides an egui-based interface that displays real-time status
//! updates from multiple audio recorders, including signal status, RMS levels,
//! and clipping indicators.

use eframe::egui;
use std::collections::HashMap;
use std::sync::{Arc, Mutex};
use std::time::{Duration, Instant};

use crate::status_reader::common::{RecorderStatus, SignalStatus};

/// Shared state for recorder statuses, thread-safe for concurrent access
pub type RecorderStatusMap = Arc<Mutex<HashMap<String, (RecorderStatus, Instant)>>>;

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
}

impl eframe::App for RecorderDisplayApp {
    fn update(&mut self, ctx: &egui::Context, _frame: &mut eframe::Frame) {
        // Set up continuous repainting to keep UI updated
        ctx.request_repaint_after(Duration::from_millis(100));

        // Clean up stale entries every 5 seconds
        if self.last_update_check.elapsed() > Duration::from_secs(5) {
            let mut statuses = self.recorder_statuses.lock().unwrap();
            let now = Instant::now();
            statuses.retain(|_, (_, last_seen)| {
                now.duration_since(*last_seen) < Duration::from_secs(30) // Remove after 30 seconds of no updates
            });
            self.last_update_check = now;
        }

        // Use larger UI scale for touch screen usability
        ctx.set_pixels_per_point(1.2);

        egui::CentralPanel::default().show(ctx, |ui| {
            // Header with larger text for readability
            ui.heading(egui::RichText::new("Session Recorder Status Display").size(24.0));
            ui.separator();

            let statuses = self.recorder_statuses.lock().unwrap();

            if statuses.is_empty() {
                ui.centered_and_justified(|ui| {
                    ui.label(egui::RichText::new("No recorders connected").size(18.0));
                });
                return;
            }

            // Create a responsive layout optimized for 800x480 display
            egui::ScrollArea::vertical().show(ui, |ui| {
                let available_width = ui.available_width();
                let card_width = 380.0; // Optimized for 800px width
                let columns = ((available_width / card_width) as usize).max(1);

                let mut current_column = 0;

                ui.horizontal_wrapped(|ui| {
                    for (_recorder_id, (status, last_seen)) in statuses.iter() {
                        if current_column >= columns {
                            current_column = 0;
                        }

                        // Status card with touch-friendly sizing
                        egui::Frame::none()
                            .fill(egui::Color32::from_gray(40))
                            .stroke(egui::Stroke::new(2.0, egui::Color32::from_gray(60)))
                            .rounding(egui::Rounding::same(12.0))
                            .inner_margin(egui::Margin::same(20.0))
                            .show(ui, |ui| {
                                ui.set_min_width(card_width - 40.0);
                                ui.set_min_height(180.0); // Ensure consistent card height

                                // Recorder name and timestamp
                                ui.horizontal(|ui| {
                                    ui.label(
                                        egui::RichText::new(&status.recorder_name)
                                            .size(22.0)
                                            .strong(),
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
                                                egui::RichText::new(format!(
                                                    "{}s ago",
                                                    age.as_secs()
                                                ))
                                                .size(14.0),
                                            );
                                        },
                                    );
                                });

                                ui.label(
                                    egui::RichText::new(format!("ID: {}", status.recorder_id))
                                        .size(14.0)
                                        .color(egui::Color32::GRAY),
                                );

                                ui.add_space(12.0);

                                // Signal status with larger icons and text
                                let (signal_text, signal_color) =
                                    match SignalStatus::try_from(status.signal_status) {
                                        Ok(SignalStatus::Signal) => {
                                            ("🟢 SIGNAL", egui::Color32::GREEN)
                                        }
                                        Ok(SignalStatus::NoSignal) => {
                                            ("🔴 NO SIGNAL", egui::Color32::RED)
                                        }
                                        Ok(SignalStatus::Unknown) | Err(_) => {
                                            ("⚪ UNKNOWN", egui::Color32::GRAY)
                                        }
                                    };

                                ui.horizontal(|ui| {
                                    ui.label(egui::RichText::new("Status:").size(16.0));
                                    ui.colored_label(
                                        signal_color,
                                        egui::RichText::new(signal_text).size(16.0).strong(),
                                    );
                                });

                                ui.add_space(8.0);

                                // RMS Level with larger progress bar
                                ui.horizontal(|ui| {
                                    ui.label(egui::RichText::new("RMS Level:").size(16.0));
                                    ui.label(
                                        egui::RichText::new(format!("{:.1}%", status.rms_percent))
                                            .size(16.0),
                                    );
                                });

                                let rms_normalized = (status.rms_percent / 100.0).clamp(0.0, 1.0);
                                let rms_color = if status.rms_percent > 80.0 {
                                    egui::Color32::RED
                                } else if status.rms_percent > 60.0 {
                                    egui::Color32::YELLOW
                                } else {
                                    egui::Color32::GREEN
                                };

                                let progress_bar = egui::ProgressBar::new(rms_normalized as f32)
                                    .fill(rms_color)
                                    .animate(true)
                                    .desired_height(20.0); // Larger progress bar for touch screens
                                ui.add(progress_bar);

                                ui.add_space(8.0);

                                // Clipping indicator with larger text and icons
                                ui.horizontal(|ui| {
                                    ui.label(egui::RichText::new("Clipping:").size(16.0));
                                    if status.clipping {
                                        ui.colored_label(
                                            egui::Color32::RED,
                                            egui::RichText::new("⚠️ YES").size(16.0),
                                        );
                                    } else {
                                        ui.colored_label(
                                            egui::Color32::GREEN,
                                            egui::RichText::new("✅ NO").size(16.0),
                                        );
                                    }
                                });
                            });

                        ui.add_space(12.0);
                        current_column += 1;
                    }
                });
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
            .with_resizable(false)
            .with_decorations(false) // No window decorations for embedded display
            .with_fullscreen(true), // Fullscreen for embedded use
        ..Default::default()
    };

    eframe::run_native(
        "Session Recorder Display",
        options,
        Box::new(|_cc| Box::new(app)),
    )
}
