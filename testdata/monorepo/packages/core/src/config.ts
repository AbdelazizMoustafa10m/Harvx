export interface Config {
  port: number;
  host: string;
  debug: boolean;
}

const defaults: Config = {
  port: 3000,
  host: "localhost",
  debug: false,
};

export function loadConfig(overrides?: Partial<Config>): Config {
  return { ...defaults, ...overrides };
}