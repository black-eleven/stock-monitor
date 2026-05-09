'use strict';

const fs = require('fs');
const path = require('path');
const config = require('./config');

function filePath(name) {
  return path.join(config.dataDir, `${name}.json`);
}

function read(name) {
  try {
    const raw = fs.readFileSync(filePath(name), 'utf-8');
    return JSON.parse(raw);
  } catch (err) {
    if (err.code === 'ENOENT') return [];
    console.error(`[storage] Corrupt JSON file: ${filePath(name)}`, err.message);
    throw err;
  }
}

function write(name, data) {
  if (!fs.existsSync(config.dataDir)) {
    fs.mkdirSync(config.dataDir, { recursive: true });
  }
  fs.writeFileSync(filePath(name), JSON.stringify(data, null, 2));
}

function readOne(name, predicate) {
  const items = read(name);
  return items.find(predicate) || null;
}

function add(name, item) {
  const items = read(name);
  items.push(item);
  write(name, items);
  return item;
}

function update(name, predicate, updater) {
  const items = read(name);
  const idx = items.findIndex(predicate);
  if (idx === -1) return null;
  items[idx] = updater(items[idx]);
  write(name, items);
  return items[idx];
}

function remove(name, predicate) {
  const items = read(name);
  const idx = items.findIndex(predicate);
  if (idx === -1) return false;
  items.splice(idx, 1);
  write(name, items);
  return true;
}

module.exports = { read, write, readOne, add, update, remove };
