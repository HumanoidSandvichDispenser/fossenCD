// SPDX-FileCopyrightText: 2026 HumanoidSandvichDispenser <humanoidsandvichdispenser@gmail.com>
//
// SPDX-License-Identifier: AGPL-3.0-or-later

//! fossenCD sync server: one always-on Teamtype peer per project.

mod api;
mod config;
mod registry;

use std::sync::Arc;

use anyhow::Result;
use config::ServerConfig;
use registry::ProjectRegistry;

#[tokio::main]
async fn main() -> Result<()> {
    tracing_subscriber::fmt()
        .with_env_filter(
            tracing_subscriber::EnvFilter::try_from_default_env()
                .unwrap_or_else(|_| "fossencd_server=info,teamtype=info".into()),
        )
        .init();

    let config = ServerConfig::from_env()?;
    let registry = Arc::new(ProjectRegistry::new(config.clone()));

    let listener = tokio::net::TcpListener::bind(config.bind_addr).await?;
    tracing::info!(addr = %config.bind_addr, storage = %config.storage_root.display(), "fossencd-server listening");

    axum::serve(listener, api::router(registry))
        .with_graceful_shutdown(shutdown_signal())
        .await?;

    Ok(())
}

async fn shutdown_signal() {
    let _ = tokio::signal::ctrl_c().await;
    tracing::info!("shutting down");
}
