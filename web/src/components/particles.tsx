"use client";

import {motion} from "framer-motion";


export function ShineEffect() {
  return (
    <motion.div
      className="absolute inset-0 overflow-hidden rounded-lg pointer-events-none"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
    >
      <motion.div
        className="absolute -inset-full bg-gradient-to-r from-transparent via-white/40 to-transparent rotate-12"
        animate={{
          x: ["-200%", "200%"],
        }}
        transition={{
          duration: 1.5,
          repeat: Number.POSITIVE_INFINITY,
          repeatDelay: 3,
          ease: "easeInOut",
        }}
      />
    </motion.div>
  );
}
