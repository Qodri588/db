const axios = require('axios');
const sqlite3 = require('sqlite3').verbose();

// Konfigurasi API Key dan URL
const api_key = '350871o0uomobcm787efod';
const base_url = "https://doodapi.com/api";

// Membuat atau membuka database SQLite
const db = new sqlite3.Database('dood.db');

// Membuat tabel folders dan files jika belum ada
db.serialize(() => {
    db.run(`CREATE TABLE IF NOT EXISTS folders (
        fld_id TEXT PRIMARY KEY,
        name TEXT,
        parent_id TEXT
    )`);

    db.run(`CREATE TABLE IF NOT EXISTS files (
        file_code TEXT PRIMARY KEY,
        title TEXT,
        download_url TEXT,
        single_img TEXT,
        length INTEGER,
        views INTEGER,
        uploaded TEXT,
        fld_id TEXT,
        name TEXT
    )`);
});

// Fungsi untuk mengambil list folder dari API Doodstream
async function getFolderList(api_key) {
    try {
        const url = `${base_url}/folder/list?key=${api_key}&fld_id=0`; // fld_id=0 untuk root folder
        const response = await axios.get(url);
        return response.data;
    } catch (error) {
        console.error("Failed to retrieve folder list:", error.message);
        return null;
    }
}

// Fungsi untuk mengambil list file dari folder tertentu
async function getFileListByFolder(api_key, folder_id) {
    try {
        const url = `${base_url}/file/list?key=${api_key}&fld_id=${folder_id}`;
        const response = await axios.get(url);
        return response.data;
    } catch (error) {
        console.error(`Failed to retrieve file list for folder ${folder_id}:`, error.message);
        return null;
    }
}

// Fungsi untuk menyimpan folder ke database
function saveFolderToDatabase(fld_id, name, parent_id) {
    return new Promise((resolve, reject) => {
        const query = `INSERT OR IGNORE INTO folders (fld_id, name, parent_id) VALUES (?, ?, ?)`;
        db.run(query, [fld_id, name, parent_id], function (err) {
            if (err) {
                return reject(err);
            }
            resolve();
        });
    });
}

// Fungsi untuk menyimpan file ke database
function saveFileToDatabase(file) {
    return new Promise((resolve, reject) => {
        const query = `INSERT OR IGNORE INTO files (file_code, title, download_url, single_img, length, views, uploaded, fld_id, name) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`;
        db.run(query, [
            file.file_code, file.title, file.download_url, file.single_img, file.length, file.views, file.uploaded, file.fld_id, file.name
        ], function (err) {
            if (err) {
                return reject(err);
            }
            resolve();
        });
    });
}

// Fungsi untuk memeriksa apakah folder sudah ada di database
function folderExists(fld_id) {
    return new Promise((resolve, reject) => {
        const query = `SELECT fld_id FROM folders WHERE fld_id = ? LIMIT 1`;
        db.get(query, [fld_id], (err, row) => {
            if (err) {
                return reject(err);
            }
            resolve(row ? true : false);
        });
    });
}

// Fungsi utama untuk mengumpulkan dan menyimpan data
async function main() {
    const folderData = await getFolderList(api_key);
    
    if (!folderData || folderData.msg !== 'OK') {
        console.log("Failed to retrieve folders.");
        return;
    }

    for (const folder of folderData.result.folders) {
        // Cek apakah folder sudah ada di database
        const exists = await folderExists(folder.fld_id);
        if (exists) {
            console.log(`Folder: ${folder.name} (already exists)`);
            continue;
        }

        // Simpan folder ke database
        await saveFolderToDatabase(folder.fld_id, folder.name, 0); // 0 untuk root folder
        console.log(`Folder: ${folder.name}`);

        // Mengambil list file dari folder tertentu
        const fileData = await getFileListByFolder(api_key, folder.fld_id);
        
        if (!fileData || fileData.msg !== 'OK') {
            console.log(`  * No files found in folder: ${folder.name}`);
            continue;
        }

        console.log("- Files:");
        for (const file of fileData.result.files) {
            // Simpan file ke database
            await saveFileToDatabase({
                file_code: file.file_code,
                title: file.title,
                download_url: file.download_url,
                single_img: file.single_img,
                length: file.length,
                views: file.views,
                uploaded: file.uploaded,
                fld_id: folder.fld_id,
                name: folder.name
            });

            // Tampilkan nama file di CLI
            console.log(`  * ${file.title}`);
        }
        console.log("**************");
    }
    
    console.log("Finished processing folders and files.");
}

// Jalankan fungsi utama
main().catch((err) => {
    console.error("An error occurred:", err.message);
});
