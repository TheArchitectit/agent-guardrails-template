export interface ContentFilterResult {
  blocked: boolean;
  matchedTopics: string[];
  severity: "none" | "warning" | "critical";
}

export interface ContentFilterConfig {
  /** Topics that are explicitly denied */
  deniedTopics?: string[];
  /** Topics that are explicitly allowed (blocks everything else) */
  allowedTopics?: string[];
  /** Strict mode: blocks any output not matching an allowed topic */
  strictMode?: boolean;
  /** Patterns for each topic (topic name -> regex patterns) */
  topicPatterns?: Record<string, string[]>;
}

const DEFAULT_TOPIC_PATTERNS: Record<string, string[]> = {
  violence: [/\b(kill|murder|attack|assault|bomb|weapon|shoot|stab)\b/i],
  hate: [/\b(hate|bigot|racist|slur|discrimination)\b/i],
  self_harm: [/\b(suicide|self.?harm|cutting|overdose)\b/i],
  sexual: [/\b(pornograph|explicit|nsfw|sexual)\b/i],
  credentials: [/\b(password|secret|token|api.?key|private.?key)\s*[:=]\s*\S+/i],
};

export class ContentFilter {
  private deniedTopics: Set<string>;
  private allowedTopics: Set<string> | null;
  private strictMode: boolean;
  private patterns: Map<string, RegExp[]>;

  constructor(config?: ContentFilterConfig) {
    this.deniedTopics = new Set(config?.deniedTopics ?? []);
    this.allowedTopics = config?.allowedTopics ? new Set(config.allowedTopics) : null;
    this.strictMode = config?.strictMode ?? false;

    this.patterns = new Map();
    const allPatterns = { ...DEFAULT_TOPIC_PATTERNS, ...config?.topicPatterns };
    for (const [topic, pats] of Object.entries(allPatterns)) {
      this.patterns.set(
        topic,
        pats.map((p: any) => (p instanceof RegExp ? p : new RegExp(p, "i"))),
      );
    }
  }

  filter(text: string): ContentFilterResult {
    if (!text || typeof text !== "string") {
      return { blocked: false, matchedTopics: [], severity: "none" };
    }

    const matchedTopics: string[] = [];

    for (const [topic, patterns] of this.patterns) {
      for (const pattern of patterns) {
        if (pattern.test(text)) {
          matchedTopics.push(topic);
          break;
        }
      }
    }

    // Check denied topics
    const deniedMatches = matchedTopics.filter((t) => this.deniedTopics.has(t));
    if (deniedMatches.length > 0) {
      return { blocked: true, matchedTopics: deniedMatches, severity: "critical" };
    }

    // Strict mode: block if not in allowed topics
    if (this.strictMode && this.allowedTopics) {
      const notAllowed = matchedTopics.filter((t) => !this.allowedTopics!.has(t));
      if (notAllowed.length > 0) {
        return { blocked: true, matchedTopics: notAllowed, severity: "warning" };
      }
    }

    if (matchedTopics.length > 0) {
      return { blocked: false, matchedTopics, severity: "warning" };
    }

    return { blocked: false, matchedTopics: [], severity: "none" };
  }

  addDeniedTopic(topic: string): void {
    this.deniedTopics.add(topic);
  }

  removeDeniedTopic(topic: string): void {
    this.deniedTopics.delete(topic);
  }
}
