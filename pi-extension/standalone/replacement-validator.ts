import * as fs from "node:fs";

export interface ReplacementValidation {
  valid: boolean;
  reason?: string;
  expectedLineCount?: number;
  actualLineCount?: number;
}

export class ExactReplacementValidator {
  validateEdit(filePath: string, oldContent: string): ReplacementValidation {
    // Check file exists
    if (!fs.existsSync(filePath)) {
      return {
        valid: false,
        reason: `File does not exist: ${filePath}`,
      };
    }

    let actualContent: string;
    try {
      actualContent = fs.readFileSync(filePath, "utf-8");
    } catch {
      return {
        valid: false,
        reason: `Cannot read file: ${filePath}`,
      };
    }

    // Check if old content exists in the file
    if (!actualContent.includes(oldContent)) {
      // Try to give a helpful error
      const oldLines = oldContent.split("\n");
      const actualLines = actualContent.split("\n");

      // Find first mismatching line
      let firstMismatch = -1;
      for (let i = 0; i < Math.min(oldLines.length, actualLines.length); i++) {
        if (oldLines[i] !== actualLines[i]) {
          firstMismatch = i;
          break;
        }
      }

      const reason = firstMismatch >= 0
        ? `Old content does not match file at line ${firstMismatch + 1}. The file may have been modified since it was last read.`
        : `Old content not found in file. The file may have been modified since it was last read.`;

      return {
        valid: false,
        reason,
        expectedLineCount: oldLines.length,
        actualLineCount: actualLines.length,
      };
    }

    return { valid: true };
  }

  validateWrite(filePath: string, expectedContent: string): ReplacementValidation {
    if (!fs.existsSync(filePath)) {
      // New file — write is always valid
      return { valid: true };
    }

    let actualContent: string;
    try {
      actualContent = fs.readFileSync(filePath, "utf-8");
    } catch {
      // Can't read — allow write (might fix it)
      return { valid: true };
    }

    // For Write, warn if overwriting with identical content
    if (actualContent === expectedContent) {
      return {
        valid: true,
        reason: "File content is identical — no change needed",
      };
    }

    return { valid: true };
  }
}
