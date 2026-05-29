//! Avahi discovery backend: talks to the system `avahi-daemon` over D-Bus.
//!
//! Uses zbus (pure-Rust D-Bus) to drive Avahi's `org.freedesktop.Avahi.Server`
//! API: it creates a `ServiceBrowser`, listens for `ItemNew`/`ItemRemove`
//! signals, and resolves each new item to an address/port via `ResolveService`.
//! Preferred on hosts already running avahi-daemon (e.g. a Raspberry Pi), so the
//! recorder doesn't run a second in-process mDNS stack.

use std::collections::HashMap;
use std::net::Ipv4Addr;
use std::time::Instant;

use anyhow::Context;
use futures_util::StreamExt;
use log::{error, info, warn};
use tokio::sync::mpsc;
use tokio::task::JoinHandle;
use zbus::zvariant::OwnedObjectPath;
use zbus::{Connection, Proxy};

use super::{DiscoveryConfig, ServiceDiscovery, ServiceEvent, ServiceInfo};

// Avahi protocol/interface constants (see avahi-common/address.h).
const AVAHI_IF_UNSPEC: i32 = -1;
const AVAHI_PROTO_UNSPEC: i32 = -1;
const AVAHI_PROTO_INET: i32 = 0; // request an IPv4 address when resolving

const AVAHI_DEST: &str = "org.freedesktop.Avahi";
const AVAHI_SERVER_IFACE: &str = "org.freedesktop.Avahi.Server";
const AVAHI_BROWSER_IFACE: &str = "org.freedesktop.Avahi.ServiceBrowser";

/// `(interface, protocol, name, type, domain, flags)` — the body of both the
/// `ItemNew` and `ItemRemove` ServiceBrowser signals.
type BrowserItem = (i32, i32, String, String, String, u32);

/// Return tuple of `Server.ResolveService`:
/// `(interface, protocol, name, type, domain, host, aprotocol, address, port, txt, flags)`.
type ResolveReply = (
    i32,
    i32,
    String,
    String,
    String,
    String,
    i32,
    String,
    u16,
    Vec<Vec<u8>>,
    u32,
);

/// Avahi (D-Bus) service discovery backend.
pub struct AvahiDiscovery {
    config: DiscoveryConfig,
    handle: Option<JoinHandle<()>>,
}

impl AvahiDiscovery {
    pub fn new(config: DiscoveryConfig) -> Self {
        Self {
            config,
            handle: None,
        }
    }
}

#[async_trait::async_trait]
impl ServiceDiscovery for AvahiDiscovery {
    async fn start(&mut self) -> anyhow::Result<mpsc::UnboundedReceiver<ServiceEvent>> {
        let (tx, rx) = mpsc::unbounded_channel();
        let (service_type, domain) = avahi_type_and_domain(&self.config.service_type);

        // All D-Bus state lives inside the task, so there are no cross-task
        // lifetimes to manage. Connection/browse errors are logged here; if the
        // avahi-daemon isn't reachable the receiver simply yields nothing.
        let handle = tokio::spawn(async move {
            if let Err(e) = run_browser(service_type, domain, tx).await {
                error!("Avahi discovery stopped: {e:#}");
            }
        });

        self.handle = Some(handle);
        Ok(rx)
    }

    async fn stop(&mut self) {
        if let Some(handle) = self.handle.take() {
            handle.abort();
            let _ = handle.await;
        }
    }
}

impl Drop for AvahiDiscovery {
    fn drop(&mut self) {
        if let Some(handle) = self.handle.take() {
            handle.abort();
        }
    }
}

/// Convert an mDNS-style service type (`_x._tcp.local.`) into the
/// `(type, domain)` pair Avahi expects (`_x._tcp`, `""` = default/local).
fn avahi_type_and_domain(service_type: &str) -> (String, String) {
    let t = service_type.strip_suffix('.').unwrap_or(service_type);
    let t = t.strip_suffix(".local").unwrap_or(t);
    (t.to_string(), String::new())
}

async fn run_browser(
    service_type: String,
    domain: String,
    tx: mpsc::UnboundedSender<ServiceEvent>,
) -> anyhow::Result<()> {
    let conn = Connection::system()
        .await
        .context("connecting to the system D-Bus")?;

    let server = Proxy::new(&conn, AVAHI_DEST, "/", AVAHI_SERVER_IFACE)
        .await
        .context("creating Avahi Server proxy")?;

    let browser_path: OwnedObjectPath = server
        .call(
            "ServiceBrowserNew",
            &(
                AVAHI_IF_UNSPEC,
                AVAHI_PROTO_UNSPEC,
                service_type.as_str(),
                domain.as_str(),
                0u32,
            ),
        )
        .await
        .context("ServiceBrowserNew")?;

    info!(
        "Avahi: browsing {} (browser {})",
        service_type,
        browser_path.as_str()
    );

    let browser = Proxy::new(&conn, AVAHI_DEST, browser_path.as_str(), AVAHI_BROWSER_IFACE)
        .await
        .context("creating ServiceBrowser proxy")?;

    let mut item_new = browser.receive_signal("ItemNew").await?;
    let mut item_remove = browser.receive_signal("ItemRemove").await?;

    // Local set of instance names we've already reported, to distinguish
    // Discovered from Updated (mirrors the mDNS backend's behaviour).
    let mut known: HashMap<String, ()> = HashMap::new();

    loop {
        tokio::select! {
            Some(msg) = item_new.next() => {
                let item: BrowserItem = match msg.body().deserialize() {
                    Ok(v) => v,
                    Err(e) => { warn!("Avahi: cannot decode ItemNew: {e}"); continue; }
                };
                let (iface, proto, name, typ, dom, _flags) = item;
                match resolve(&server, iface, proto, &name, &typ, &dom).await {
                    Ok(info) => {
                        let event = if known.insert(name.clone(), ()).is_some() {
                            ServiceEvent::ServiceUpdated(info)
                        } else {
                            ServiceEvent::ServiceDiscovered(info)
                        };
                        if tx.send(event).is_err() {
                            break; // receiver dropped
                        }
                    }
                    Err(e) => warn!("Avahi: resolve of {name} failed: {e:#}"),
                }
            }
            Some(msg) = item_remove.next() => {
                let item: BrowserItem = match msg.body().deserialize() {
                    Ok(v) => v,
                    Err(e) => { warn!("Avahi: cannot decode ItemRemove: {e}"); continue; }
                };
                let name = item.2;
                known.remove(&name);
                if tx.send(ServiceEvent::ServiceRemoved(name)).is_err() {
                    break;
                }
            }
            else => break,
        }
    }

    Ok(())
}

async fn resolve(
    server: &Proxy<'_>,
    iface: i32,
    proto: i32,
    name: &str,
    typ: &str,
    domain: &str,
) -> anyhow::Result<ServiceInfo> {
    let reply: ResolveReply = server
        .call(
            "ResolveService",
            &(iface, proto, name, typ, domain, AVAHI_PROTO_INET, 0u32),
        )
        .await
        .context("ResolveService")?;

    let (_iface, _proto, name, _typ, _domain, host, _aproto, address, port, txt, _flags) = reply;

    let addr: Ipv4Addr = address
        .parse()
        .with_context(|| format!("resolved address {address} is not IPv4"))?;

    let mut properties = HashMap::new();
    for entry in txt {
        if let Ok(s) = String::from_utf8(entry) {
            match s.split_once('=') {
                Some((k, v)) => properties.insert(k.to_string(), v.to_string()),
                None => properties.insert(s, String::new()),
            };
        }
    }

    Ok(ServiceInfo {
        instance_name: name,
        hostname: host,
        addresses: vec![addr],
        port,
        properties,
        discovered_at: Instant::now(),
        last_seen: Instant::now(),
    })
}

#[cfg(test)]
mod tests {
    use super::avahi_type_and_domain;

    #[test]
    fn strips_local_domain_and_trailing_dot() {
        let (typ, domain) =
            avahi_type_and_domain("_session-recorder-chunksink._tcp.local.");
        assert_eq!(typ, "_session-recorder-chunksink._tcp");
        assert_eq!(domain, "");
    }

    #[test]
    fn leaves_bare_type_untouched() {
        let (typ, domain) = avahi_type_and_domain("_foo._tcp");
        assert_eq!(typ, "_foo._tcp");
        assert_eq!(domain, "");
    }
}
