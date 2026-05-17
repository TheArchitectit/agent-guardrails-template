import { spawn, type ChildProcess } from "node:child_process";

export interface SandboxConfig {
  /** Docker image to use (default: alpine:latest) */
  image?: string;
  /** Mount paths as read-only */
  readOnlyMounts?: string[];
  /** Mount paths as read-write */
  readWriteMounts?: string[];
  /** Network access (default: none) */
  networkAccess?: boolean;
  /** Memory limit (e.g., "512m") */
  memoryLimit?: string;
  /** CPU limit (e.g., "1.0") */
  cpuLimit?: string;
  /** Timeout in milliseconds (default: 30000) */
  timeout?: number;
  /** Working directory inside container */
  workdir?: string;
}

export interface SandboxResult {
  exitCode: number | null;
  stdout: string;
  stderr: string;
  timedOut: boolean;
}

export class SandboxRunner {
  private config: Required<Pick<SandboxConfig, "image" | "timeout">> &
    Pick<SandboxConfig, "readOnlyMounts" | "readWriteMounts" | "networkAccess" | "memoryLimit" | "cpuLimit" | "workdir">;

  constructor(config?: SandboxConfig) {
    this.config = {
      image: config?.image ?? "alpine:latest",
      timeout: config?.timeout ?? 30000,
      readOnlyMounts: config?.readOnlyMounts,
      readWriteMounts: config?.readWriteMounts,
      networkAccess: config?.networkAccess,
      memoryLimit: config?.memoryLimit,
      cpuLimit: config?.cpuLimit,
      workdir: config?.workdir,
    };
  }

  async isAvailable(): Promise<boolean> {
    return new Promise((resolve) => {
      const proc = spawn("docker", ["--version"], { stdio: "ignore" });
      proc.on("error", () => resolve(false));
      proc.on("exit", (code) => resolve(code === 0));
    });
  }

  async run(command: string[], options?: Partial<SandboxConfig>): Promise<SandboxResult> {
    const timeout = options?.timeout ?? this.config.timeout;
    const image = options?.image ?? this.config.image;

    const dockerArgs: string[] = ["run", "--rm"];

    // No network by default
    if (!(options?.networkAccess ?? this.config.networkAccess)) {
      dockerArgs.push("--network=none");
    }

    // Resource limits
    const mem = options?.memoryLimit ?? this.config.memoryLimit;
    if (mem) dockerArgs.push(`--memory=${mem}`);
    const cpu = options?.cpuLimit ?? this.config.cpuLimit;
    if (cpu) dockerArgs.push(`--cpus=${cpu}`);

    // Read-only mounts
    const roMounts = options?.readOnlyMounts ?? this.config.readOnlyMounts;
    if (roMounts) {
      for (const mount of roMounts) {
        dockerArgs.push("-v", `${mount}:${mount}:ro`);
      }
    }

    // Read-write mounts
    const rwMounts = options?.readWriteMounts ?? this.config.readWriteMounts;
    if (rwMounts) {
      for (const mount of rwMounts) {
        dockerArgs.push("-v", `${mount}:${mount}:rw`);
      }
    }

    // Workdir
    const workdir = options?.workdir ?? this.config.workdir;
    if (workdir) dockerArgs.push("-w", workdir);

    dockerArgs.push(image, ...command);

    return new Promise((resolve) => {
      const proc = spawn("docker", dockerArgs);
      let stdout = "";
      let stderr = "";
      let timedOut = false;
      let timer: ReturnType<typeof setTimeout> | null = null;

      proc.stdout?.on("data", (data: Buffer) => {
        stdout += data.toString();
      });

      proc.stderr?.on("data", (data: Buffer) => {
        stderr += data.toString();
      });

      timer = setTimeout(() => {
        timedOut = true;
        proc.kill("SIGKILL");
      }, timeout);

      proc.on("close", (code) => {
        if (timer) clearTimeout(timer);
        resolve({ exitCode: code, stdout, stderr, timedOut });
      });

      proc.on("error", (err) => {
        if (timer) clearTimeout(timer);
        stderr += err.message;
        resolve({ exitCode: null, stdout, stderr, timedOut: false });
      });
    });
  }
}
