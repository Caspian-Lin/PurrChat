// Prevents additional console window on Windows in release, DO NOT REMOVE!!
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use std::fs::File;
use std::io::Write;

fn main() {
    // 构建应用
    let result = tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .run(tauri::generate_context!());

    // 检查运行结果
    if let Err(e) = result {
        // 如果出错，将具体错误信息写入桌面文件
        let err_msg = format!("Application failed to start.\nError: {}", e);
        // 请确保路径正确，28491 是你日志里的用户名
        let path = "C:\\Users\\28491\\Desktop\\startup_error.txt"; 
        
        if let Ok(mut file) = File::create(path) {
            let _ = file.write_all(err_msg.as_bytes());
        }
    }
}