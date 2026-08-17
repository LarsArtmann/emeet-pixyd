export const featureIconKeys = [
  "eye",
  "shield",
  "volume",
  "globe",
  "bolt",
  "chart",
  "plug",
  "terminal",
] as const;
export type FeatureIcon = (typeof featureIconKeys)[number];

export interface Feature {
  icon: FeatureIcon;
  title: string;
  desc: string;
}

export interface StepCard {
  step: string;
  stepColor: "accent" | "amber";
  title: string;
  desc: string;
  code?: string;
}

export type ComparisonVariant = "Manual" | "Browser extension" | "emeet-pixyd";

export interface ComparisonItem {
  variant: ComparisonVariant;
  pros: string[];
  cons: string[];
  accent: boolean;
}

export type MatrixValue = "yes" | "no" | string;

export interface MatrixRow {
  feature: string;
  values: [MatrixValue, MatrixValue, MatrixValue];
}

export interface ComparisonMatrix {
  columns: [ComparisonVariant, ComparisonVariant, ComparisonVariant];
  rows: MatrixRow[];
}

export const useCaseIconKeys = ["video", "chat", "stream", "mic", "shield", "camera"] as const;
export type UseCaseIcon = (typeof useCaseIconKeys)[number];

export interface UseCase {
  title: string;
  desc: string;
  icon: UseCaseIcon;
}

export const uiIconKeys = [
  "arrow-external",
  "arrow-right",
  "github",
  "menu",
  "close",
  "sun",
  "moon",
  "star",
] as const;
export type UIIcon = (typeof uiIconKeys)[number];

export type IconName = FeatureIcon | UseCaseIcon | UIIcon;
