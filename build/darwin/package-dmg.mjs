#!/usr/bin/env node
/**
 * Builds a styled macOS DMG using the same appdmg layout as create-dmg,
 * with a MetalTracker-branded background image.
 *
 * @see https://github.com/sindresorhus/create-dmg
 */
import process from 'node:process';
import path from 'node:path';
import fs from 'node:fs';
import {fileURLToPath} from 'node:url';
import {Resvg} from '@resvg/resvg-js';
import appdmg from 'appdmg';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const backgroundSvgPath = path.join(__dirname, 'dmg-background.svg');
const backgroundPath = path.join(__dirname, 'dmg-background.png');
const backgroundRetinaPath = path.join(__dirname, 'dmg-background@2x.png');
const backgroundWidth = 660;
const backgroundHeight = 400;

function printUsage() {
  console.error(`Usage: node package-dmg.mjs <app-path> <output-dmg-path> [--dmg-title=<title>]

Options:
  --dmg-title=<title>  Window title (max 27 characters, default: MetalTracker)
`);
}

function parseArguments(argv) {
  const positional = [];
  let dmgTitle = 'MetalTracker';

  for (const argument of argv) {
    if (argument.startsWith('--dmg-title=')) {
      dmgTitle = argument.slice('--dmg-title='.length);
      continue;
    }

    if (argument.startsWith('-')) {
      console.error(`Unknown option: ${argument}`);
      printUsage();
      process.exit(1);
    }

    positional.push(argument);
  }

  return {positional, dmgTitle};
}

function renderBackgroundPng({svgPath, outputPath, width, height}) {
  const svg = fs.readFileSync(svgPath);
  const resvg = new Resvg(svg, {
    fitTo: {
      mode: 'width',
      value: width,
    },
  });
  const rendered = resvg.render();
  const png = rendered.asPng();

  if (rendered.height !== height) {
    throw new Error(`Expected background height ${height}, got ${rendered.height}`);
  }

  fs.writeFileSync(outputPath, png);
}

function ensureBackgroundImages() {
  if (!fs.existsSync(backgroundSvgPath)) {
    throw new Error(`Background SVG not found: ${backgroundSvgPath}`);
  }

  renderBackgroundPng({
    svgPath: backgroundSvgPath,
    outputPath: backgroundPath,
    width: backgroundWidth,
    height: backgroundHeight,
  });
  renderBackgroundPng({
    svgPath: backgroundSvgPath,
    outputPath: backgroundRetinaPath,
    width: backgroundWidth * 2,
    height: backgroundHeight * 2,
  });
}

function buildDiskImage({appPath, outputPath, dmgTitle}) {
  if (process.platform !== 'darwin') {
    throw new Error('DMG packaging requires macOS');
  }

  if (!fs.existsSync(appPath)) {
    throw new Error(`App bundle not found: ${appPath}`);
  }

  if (dmgTitle.length > 27) {
    throw new Error('The disk image title cannot exceed 27 characters');
  }

  fs.mkdirSync(path.dirname(outputPath), {recursive: true});

  if (fs.existsSync(outputPath)) {
    fs.unlinkSync(outputPath);
  }

  return new Promise((resolve, reject) => {
    const eventEmitter = appdmg({
      target: outputPath,
      basepath: process.cwd(),
      specification: {
        title: dmgTitle,
        background: backgroundPath,
        'icon-size': 160,
        format: 'ULFO',
        filesystem: 'APFS',
        window: {
          size: {
            width: backgroundWidth,
            height: backgroundHeight,
          },
        },
        contents: [
          {
            x: 180,
            y: 152,
            type: 'file',
            path: appPath,
          },
          {
            x: 480,
            y: 152,
            type: 'link',
            path: '/Applications',
          },
        ],
      },
    });

    eventEmitter.on('finish', resolve);
    eventEmitter.on('error', reject);
  });
}

const {positional, dmgTitle} = parseArguments(process.argv.slice(2));
const [appPath, outputPath] = positional;

if (!appPath || !outputPath) {
  printUsage();
  process.exit(1);
}

const resolvedAppPath = path.resolve(appPath);
const resolvedOutputPath = path.resolve(outputPath);

try {
  console.log('Rendering DMG background...');
  ensureBackgroundImages();
  console.log(`Creating DMG: ${resolvedOutputPath}`);
  await buildDiskImage({
    appPath: resolvedAppPath,
    outputPath: resolvedOutputPath,
    dmgTitle,
  });
  console.log(`Created "${path.basename(resolvedOutputPath)}"`);
} catch (error) {
  console.error(error instanceof Error ? error.message : error);
  process.exit(1);
}
