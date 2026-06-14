// SPDX-FileCopyrightText: 2026 HumanoidSandvichDispenser <humanoidsandvichdispenser@gmail.com>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

use std::net::SocketAddr;
use std::path::PathBuf;

use anyhow::{Context, Result};

#[derive(Clone, Debug)]
pub struct ServerConfig {
    pub bind_addr: SocketAddr,
    // each project lives at <storage_root>/<id>
    pub storage_root: PathBuf,
    pub iroh_relay: Option<String>,
    pub magic_wormhole_relay: Option<String>,
}

impl ServerConfig {
    pub fn from_env() -> Result<Self> {
        let bind_addr = std::env::var("FOSSENCD_BIND")
            .unwrap_or_else(|_| "127.0.0.1:8787".to_string())
            .parse()
            .context("FOSSENCD_BIND is not a valid socket address")?;

        let storage_root = std::env::var("FOSSENCD_STORAGE")
            .unwrap_or_else(|_| "./projects".to_string())
            .into();

        Ok(Self {
            bind_addr,
            storage_root,
            iroh_relay: std::env::var("FOSSENCD_IROH_RELAY").ok(),
            magic_wormhole_relay: std::env::var("FOSSENCD_WORMHOLE_RELAY").ok(),
        })
    }
}
