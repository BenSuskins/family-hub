"use client";

import dynamic from "next/dynamic";

const DisplayClient = dynamic(() => import("@/components/display-client").then(module => module.DisplayClient), { ssr: false });

export default function DisplayPage() {
  return <DisplayClient />;
}
