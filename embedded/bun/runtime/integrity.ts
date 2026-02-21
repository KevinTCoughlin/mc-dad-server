// Script integrity verification using SHA-256 manifests.

import { join } from "node:path";

interface Manifest {
  [filename: string]: string;
}

export class IntegrityChecker {
  /** Compute SHA-256 hash of a file. */
  private async hashFile(path: string): Promise<string> {
    const data = await Bun.file(path).arrayBuffer();
    const hasher = new Bun.CryptoHasher("sha256");
    hasher.update(data);
    return hasher.digest("hex") as string;
  }

  /** Verify scripts against .manifest.json. Returns list of modified files. */
  async verify(scriptsDir: string, files: string[]): Promise<string[]> {
    const manifestPath = join(scriptsDir, ".manifest.json");
    const manifestFile = Bun.file(manifestPath);
    let manifest: Manifest;

    if (!(await manifestFile.exists())) {
      console.log("[mc-scripts] No manifest found, generating initial manifest");
      await this.regenerate(scriptsDir, files);
      return [];
    }

    try {
      manifest = await manifestFile.json();
    } catch {
      console.warn("[mc-scripts] WARNING: Corrupt manifest, regenerating");
      await this.regenerate(scriptsDir, files);
      return [];
    }

    const mismatched: string[] = [];
    const currentFiles = new Set(files);

    for (const file of files) {
      const hash = await this.hashFile(join(scriptsDir, file));
      if (manifest[file] === undefined) {
        console.log(`[mc-scripts] New script not in manifest: ${file}`);
      } else if (manifest[file] !== hash) {
        mismatched.push(file);
      }
    }

    // Check for files in manifest but missing on disk
    for (const file of Object.keys(manifest)) {
      if (!currentFiles.has(file)) {
        console.log(`[mc-scripts] Script in manifest but missing from disk: ${file}`);
      }
    }

    return mismatched;
  }

  /** Regenerate .manifest.json from current file state. */
  async regenerate(scriptsDir: string, files: string[]): Promise<void> {
    const manifest: Manifest = {};
    for (const file of files) {
      manifest[file] = await this.hashFile(join(scriptsDir, file));
    }
    const manifestPath = join(scriptsDir, ".manifest.json");
    await Bun.write(manifestPath, JSON.stringify(manifest, null, 2) + "\n");
    console.log(`[mc-scripts] Manifest written with ${files.length} entries`);
  }
}
